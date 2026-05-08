package native

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultDispatcherQueueSize       = 64
	DefaultDispatcherTimeout         = 5 * time.Second
	DefaultDispatcherMaxPayloadBytes = 1 << 20
)

type DispatchErrorCode string

const (
	DispatchErrorInvalidRequest   DispatchErrorCode = "invalid_request"
	DispatchErrorUnauthorized     DispatchErrorCode = "unauthorized"
	DispatchErrorBackpressure     DispatchErrorCode = "backpressure"
	DispatchErrorClosed           DispatchErrorCode = "dispatcher_closed"
	DispatchErrorTimeout          DispatchErrorCode = "timeout"
	DispatchErrorCanceled         DispatchErrorCode = "canceled"
	DispatchErrorHandlerNotFound  DispatchErrorCode = "handler_not_found"
	DispatchErrorHandlerFailed    DispatchErrorCode = "handler_failed"
	DispatchErrorHandlerPanic     DispatchErrorCode = "handler_panic"
	DispatchErrorPayloadTooLarge  DispatchErrorCode = "payload_too_large"
	DispatchErrorResponseTooLarge DispatchErrorCode = "response_too_large"
)

type DispatchError struct {
	Code    DispatchErrorCode `json:"code"`
	Message string            `json:"message"`
	Detail  string            `json:"detail,omitempty"`
}

func (e *DispatchError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

type DispatchRequest struct {
	ID         uint64
	BrowserID  int64
	FrameID    int64
	Origin     string
	Capability string
	Method     string
	Payload    []byte
	Timeout    time.Duration
}

type DispatchResponse struct {
	RequestID  uint64
	BrowserID  int64
	FrameID    int64
	Capability string
	Method     string
	Payload    []byte
	Error      *DispatchError
	Duration   time.Duration
}

type DispatchHandler func(context.Context, DispatchRequest) ([]byte, error)

type DispatchPolicy interface {
	Authorize(context.Context, DispatchRequest) error
}

type DispatchPolicyFunc func(context.Context, DispatchRequest) error

func (f DispatchPolicyFunc) Authorize(ctx context.Context, req DispatchRequest) error {
	return f(ctx, req)
}

type CapabilityMethod struct {
	Capability string
	Method     string
}

type StaticDispatchPolicy struct {
	AllowedOrigins map[string]bool
	AllowedMethods map[CapabilityMethod]bool
}

func (p StaticDispatchPolicy) Authorize(_ context.Context, req DispatchRequest) error {
	if len(p.AllowedOrigins) > 0 && !p.AllowedOrigins[req.Origin] {
		return newDispatchError(DispatchErrorUnauthorized, fmt.Sprintf("origin %q is not allowed", req.Origin))
	}
	if len(p.AllowedMethods) > 0 && !p.AllowedMethods[CapabilityMethod{Capability: req.Capability, Method: req.Method}] {
		return newDispatchError(DispatchErrorUnauthorized, fmt.Sprintf("capability method %s.%s is not allowed", req.Capability, req.Method))
	}
	return nil
}

type DispatcherOptions struct {
	QueueSize        int
	DefaultTimeout   time.Duration
	MaxPayloadBytes  int
	MaxResponseBytes int
	BorrowPayload    bool
	Policy           DispatchPolicy
}

type Dispatcher struct {
	mu               sync.RWMutex
	handlers         map[CapabilityMethod]DispatchHandler
	queue            chan dispatchWork
	done             chan struct{}
	closed           atomic.Bool
	nextID           atomic.Uint64
	wg               sync.WaitGroup
	defaultTimeout   time.Duration
	maxPayloadBytes  int
	maxResponseBytes int
	borrowPayload    bool
	policy           DispatchPolicy
}

type dispatchWork struct {
	ctx      context.Context
	req      DispatchRequest
	handler  DispatchHandler
	result   chan<- DispatchResponse
	enqueued time.Time
}

func NewDispatcher(opts DispatcherOptions) *Dispatcher {
	queueSize := opts.QueueSize
	if queueSize <= 0 {
		queueSize = DefaultDispatcherQueueSize
	}
	defaultTimeout := opts.DefaultTimeout
	if defaultTimeout <= 0 {
		defaultTimeout = DefaultDispatcherTimeout
	}
	maxPayloadBytes := opts.MaxPayloadBytes
	if maxPayloadBytes <= 0 {
		maxPayloadBytes = DefaultDispatcherMaxPayloadBytes
	}
	maxResponseBytes := opts.MaxResponseBytes
	if maxResponseBytes <= 0 {
		maxResponseBytes = maxPayloadBytes
	}
	d := &Dispatcher{
		handlers:         make(map[CapabilityMethod]DispatchHandler),
		queue:            make(chan dispatchWork, queueSize),
		done:             make(chan struct{}),
		defaultTimeout:   defaultTimeout,
		maxPayloadBytes:  maxPayloadBytes,
		maxResponseBytes: maxResponseBytes,
		borrowPayload:    opts.BorrowPayload,
		policy:           opts.Policy,
	}
	d.wg.Add(1)
	go d.run()
	return d
}

func (d *Dispatcher) Register(capability, method string, handler DispatchHandler) error {
	key, err := dispatchKey(capability, method)
	if err != nil {
		return err
	}
	if handler == nil {
		return newDispatchError(DispatchErrorInvalidRequest, "handler is required")
	}
	if d.closed.Load() {
		return newDispatchError(DispatchErrorClosed, "dispatcher is shutting down")
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.handlers[key]; exists {
		return newDispatchError(DispatchErrorInvalidRequest, fmt.Sprintf("handler already registered for %s.%s", capability, method))
	}
	d.handlers[key] = handler
	return nil
}

func (d *Dispatcher) Unregister(capability, method string) error {
	key, err := dispatchKey(capability, method)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.handlers, key)
	return nil
}

func (d *Dispatcher) Dispatch(ctx context.Context, req DispatchRequest) (<-chan DispatchResponse, error) {
	return d.enqueue(ctx, req, true)
}

func (d *Dispatcher) TryDispatch(ctx context.Context, req DispatchRequest) (<-chan DispatchResponse, error) {
	return d.enqueue(ctx, req, false)
}

func (d *Dispatcher) Invoke(ctx context.Context, req DispatchRequest) (DispatchResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, err := d.Dispatch(ctx, req)
	if err != nil {
		return DispatchResponse{}, err
	}
	select {
	case resp := <-result:
		if resp.Error != nil {
			return resp, resp.Error
		}
		return resp, nil
	case <-ctx.Done():
		err := errorFromContext(ctx.Err())
		return DispatchResponse{RequestID: req.ID, Error: err}, err
	case <-d.done:
		err := newDispatchError(DispatchErrorClosed, "dispatcher is shutting down")
		return DispatchResponse{RequestID: req.ID, Error: err}, err
	}
}

func (d *Dispatcher) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if d.closed.CompareAndSwap(false, true) {
		close(d.done)
	}
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		d.rejectQueued()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Dispatcher) enqueue(ctx context.Context, req DispatchRequest, waitForQueue bool) (<-chan DispatchResponse, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if d.closed.Load() {
		return nil, newDispatchError(DispatchErrorClosed, "dispatcher is shutting down")
	}
	normalized, handler, err := d.prepare(ctx, req)
	if err != nil {
		return nil, err
	}
	result := make(chan DispatchResponse, 1)
	work := dispatchWork{
		ctx:      ctx,
		req:      normalized,
		handler:  handler,
		result:   result,
		enqueued: time.Now(),
	}
	if waitForQueue {
		select {
		case d.queue <- work:
			return result, nil
		case <-ctx.Done():
			return nil, errorFromContext(ctx.Err())
		case <-d.done:
			return nil, newDispatchError(DispatchErrorClosed, "dispatcher is shutting down")
		}
	}
	select {
	case d.queue <- work:
		return result, nil
	case <-ctx.Done():
		return nil, errorFromContext(ctx.Err())
	case <-d.done:
		return nil, newDispatchError(DispatchErrorClosed, "dispatcher is shutting down")
	default:
		return nil, newDispatchError(DispatchErrorBackpressure, "dispatcher queue is full")
	}
}

func (d *Dispatcher) prepare(ctx context.Context, req DispatchRequest) (DispatchRequest, DispatchHandler, error) {
	if req.ID == 0 {
		req.ID = d.nextID.Add(1)
	}
	key, err := dispatchKey(req.Capability, req.Method)
	if err != nil {
		return DispatchRequest{}, nil, err
	}
	if len(req.Payload) > d.maxPayloadBytes {
		return DispatchRequest{}, nil, newDispatchError(DispatchErrorPayloadTooLarge, fmt.Sprintf("payload has %d bytes; limit is %d", len(req.Payload), d.maxPayloadBytes))
	}
	if !d.borrowPayload && len(req.Payload) > 0 {
		req.Payload = append([]byte(nil), req.Payload...)
	}

	d.mu.RLock()
	handler, ok := d.handlers[key]
	d.mu.RUnlock()
	if !ok {
		return DispatchRequest{}, nil, newDispatchError(DispatchErrorHandlerNotFound, fmt.Sprintf("handler not registered for %s.%s", req.Capability, req.Method))
	}
	if d.policy != nil {
		if err := d.policy.Authorize(ctx, req); err != nil {
			return DispatchRequest{}, nil, normalizeDispatchError(err)
		}
	}
	return req, handler, nil
}

func (d *Dispatcher) run() {
	defer d.wg.Done()
	for {
		select {
		case work := <-d.queue:
			d.handle(work)
		case <-d.done:
			return
		}
	}
}

func (d *Dispatcher) handle(work dispatchWork) {
	start := time.Now()
	ctx, cancel := d.handlerContext(work.ctx, work.req.Timeout)
	defer cancel()

	resp := DispatchResponse{
		RequestID:  work.req.ID,
		BrowserID:  work.req.BrowserID,
		FrameID:    work.req.FrameID,
		Capability: work.req.Capability,
		Method:     work.req.Method,
	}
	payload, err := safeDispatchCall(ctx, work.handler, work.req)
	resp.Duration = time.Since(start)
	if err != nil {
		resp.Error = normalizeDispatchError(err)
		work.result <- resp
		return
	}
	if len(payload) > d.maxResponseBytes {
		resp.Error = newDispatchError(DispatchErrorResponseTooLarge, fmt.Sprintf("response has %d bytes; limit is %d", len(payload), d.maxResponseBytes))
		work.result <- resp
		return
	}
	resp.Payload = payload
	work.result <- resp
}

func (d *Dispatcher) handlerContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		timeout = d.defaultTimeout
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	go func() {
		select {
		case <-d.done:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, cancel
}

func safeDispatchCall(ctx context.Context, handler DispatchHandler, req DispatchRequest) (payload []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &DispatchError{
				Code:    DispatchErrorHandlerPanic,
				Message: fmt.Sprintf("handler panic: %v", recovered),
				Detail:  string(debug.Stack()),
			}
		}
	}()
	return handler(ctx, req)
}

func (d *Dispatcher) rejectQueued() {
	err := newDispatchError(DispatchErrorClosed, "dispatcher is shut down")
	for {
		select {
		case work := <-d.queue:
			work.result <- DispatchResponse{
				RequestID:  work.req.ID,
				BrowserID:  work.req.BrowserID,
				FrameID:    work.req.FrameID,
				Capability: work.req.Capability,
				Method:     work.req.Method,
				Error:      err,
			}
		default:
			return
		}
	}
}

func dispatchKey(capability, method string) (CapabilityMethod, error) {
	if capability == "" {
		return CapabilityMethod{}, newDispatchError(DispatchErrorInvalidRequest, "capability is required")
	}
	if method == "" {
		return CapabilityMethod{}, newDispatchError(DispatchErrorInvalidRequest, "method is required")
	}
	return CapabilityMethod{Capability: capability, Method: method}, nil
}

func normalizeDispatchError(err error) *DispatchError {
	if err == nil {
		return nil
	}
	var dispatchErr *DispatchError
	if errors.As(err, &dispatchErr) {
		return dispatchErr
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return newDispatchError(DispatchErrorTimeout, "request timed out")
	}
	if errors.Is(err, context.Canceled) {
		return newDispatchError(DispatchErrorCanceled, "request was canceled")
	}
	return &DispatchError{
		Code:    DispatchErrorHandlerFailed,
		Message: err.Error(),
	}
}

func errorFromContext(err error) *DispatchError {
	return normalizeDispatchError(err)
}

func newDispatchError(code DispatchErrorCode, message string) *DispatchError {
	return &DispatchError{
		Code:    code,
		Message: message,
	}
}
