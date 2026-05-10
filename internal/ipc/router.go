package ipc

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var (
	ErrChannelRequired = errors.New("channel is required")
	ErrHandlerRequired = errors.New("handler is required")
	ErrHandlerExists   = errors.New("handler already registered")
	ErrNoHandler       = errors.New("handler not registered")
	ErrPortClosed      = errors.New("message port is closed")
)

type Message struct {
	Channel string
	FrameID int64
	Args    []any
}

type Reply struct {
	Value any
}

type Handler func(context.Context, Message) (Reply, error)

type MessageEvent struct {
	Data  any
	Ports []*MessagePort
}

type MessagePort struct {
	mu       sync.Mutex
	peer     *MessagePort
	started  bool
	closed   bool
	queue    []MessageEvent
	handlers []func(MessageEvent)
}

type Router struct {
	mu       sync.RWMutex
	handlers map[string]handlerEntry
}

type handlerEntry struct {
	handler Handler
	once    bool
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]handlerEntry)}
}

func NewMessageChannel() (*MessagePort, *MessagePort) {
	port1 := &MessagePort{}
	port2 := &MessagePort{}
	port1.peer = port2
	port2.peer = port1
	return port1, port2
}

func (r *Router) Register(channel string, handler Handler) error {
	return r.Handle(channel, handler)
}

func (r *Router) Handle(channel string, handler Handler) error {
	return r.handle(channel, handler, false)
}

func (r *Router) HandleOnce(channel string, handler Handler) error {
	return r.handle(channel, handler, true)
}

func (r *Router) RemoveHandler(channel string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, channel)
}

func (r *Router) HasHandler(channel string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[channel]
	return ok
}

func (r *Router) handle(channel string, handler Handler, once bool) error {
	if channel == "" {
		return ErrChannelRequired
	}
	if handler == nil {
		return ErrHandlerRequired
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[channel]; exists {
		return fmt.Errorf("%w: %s", ErrHandlerExists, channel)
	}
	r.handlers[channel] = handlerEntry{handler: handler, once: once}
	return nil
}

func (r *Router) Invoke(ctx context.Context, msg Message) (Reply, error) {
	if msg.Channel == "" {
		return Reply{}, ErrChannelRequired
	}

	handler, ok := r.takeHandler(msg.Channel)
	if !ok {
		return Reply{}, fmt.Errorf("%w: %s", ErrNoHandler, msg.Channel)
	}

	return handler(ctx, msg)
}

func (r *Router) takeHandler(channel string) (Handler, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.handlers[channel]
	if !ok {
		return nil, false
	}
	if entry.once {
		delete(r.handlers, channel)
	}
	return entry.handler, true
}

func (p *MessagePort) Start() {
	p.mu.Lock()
	if p.closed || p.started {
		p.mu.Unlock()
		return
	}
	p.started = true
	queued := append([]MessageEvent(nil), p.queue...)
	p.queue = nil
	handlers := append([]func(MessageEvent){}, p.handlers...)
	p.mu.Unlock()

	for _, event := range queued {
		for _, handler := range handlers {
			handler(event)
		}
	}
}

func (p *MessagePort) OnMessage(handler func(MessageEvent)) {
	if handler == nil {
		return
	}
	p.mu.Lock()
	p.handlers = append(p.handlers, handler)
	p.mu.Unlock()
}

func (p *MessagePort) PostMessage(data any, ports ...*MessagePort) error {
	p.mu.Lock()
	if p.closed || p.peer == nil {
		p.mu.Unlock()
		return ErrPortClosed
	}
	peer := p.peer
	p.mu.Unlock()

	peer.deliver(MessageEvent{Data: data, Ports: append([]*MessagePort(nil), ports...)})
	return nil
}

func (p *MessagePort) Close() {
	p.mu.Lock()
	p.closed = true
	p.queue = nil
	p.handlers = nil
	p.mu.Unlock()
}

func (p *MessagePort) deliver(event MessageEvent) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	if !p.started {
		p.queue = append(p.queue, event)
		p.mu.Unlock()
		return
	}
	handlers := append([]func(MessageEvent){}, p.handlers...)
	p.mu.Unlock()

	for _, handler := range handlers {
		handler(event)
	}
}
