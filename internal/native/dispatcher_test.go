package native

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestDispatcherInvokeRoutesRequestMetadata(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	defer mustShutdownDispatcher(t, dispatcher)

	err := dispatcher.Register("files", "readText", func(_ context.Context, req DispatchRequest) ([]byte, error) {
		if req.ID == 0 {
			t.Fatal("request ID was not assigned")
		}
		if req.BrowserID != 7 {
			t.Fatalf("BrowserID = %d, want 7", req.BrowserID)
		}
		if req.FrameID != 11 {
			t.Fatalf("FrameID = %d, want 11", req.FrameID)
		}
		if req.Origin != "app://fixture" {
			t.Fatalf("Origin = %q, want app://fixture", req.Origin)
		}
		return append([]byte("read:"), req.Payload...), nil
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	resp, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		BrowserID:  7,
		FrameID:    11,
		Origin:     "app://fixture",
		Capability: "files",
		Method:     "readText",
		Payload:    []byte("notes.txt"),
	})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if resp.RequestID == 0 {
		t.Fatal("Response RequestID was not assigned")
	}
	if string(resp.Payload) != "read:notes.txt" {
		t.Fatalf("Payload = %q, want read:notes.txt", resp.Payload)
	}
}

func TestDispatcherCopiesPayloadByDefault(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	defer mustShutdownDispatcher(t, dispatcher)

	gate := make(chan struct{})
	if err := dispatcher.Register("clipboard", "writeText", func(_ context.Context, req DispatchRequest) ([]byte, error) {
		<-gate
		return req.Payload, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	payload := []byte("abc")
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		Capability: "clipboard",
		Method:     "writeText",
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	payload[0] = 'z'
	close(gate)

	resp := <-result
	if resp.Error != nil {
		t.Fatalf("Dispatch response error = %v", resp.Error)
	}
	if string(resp.Payload) != "abc" {
		t.Fatalf("Payload = %q, want copied payload abc", resp.Payload)
	}
}

func TestDispatcherRejectsOversizedPayload(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{MaxPayloadBytes: 4})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("files", "readBytes", func(context.Context, DispatchRequest) ([]byte, error) {
		t.Fatal("handler should not run")
		return nil, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Capability: "files",
		Method:     "readBytes",
		Payload:    []byte("too-large"),
	})
	assertDispatchCode(t, err, DispatchErrorPayloadTooLarge)
}

func TestDispatcherRejectsOversizedResponse(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{MaxResponseBytes: 4})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("files", "readBytes", func(context.Context, DispatchRequest) ([]byte, error) {
		return []byte("too-large"), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Capability: "files",
		Method:     "readBytes",
	})
	assertDispatchCode(t, err, DispatchErrorResponseTooLarge)
}

func TestDispatcherAppliesStaticPolicy(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{
		Policy: StaticDispatchPolicy{
			AllowedOrigins: map[string]bool{"app://fixture": true},
			AllowedMethods: map[CapabilityMethod]bool{{Capability: "dialog", Method: "openFile"}: true},
		},
	})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("dialog", "openFile", func(context.Context, DispatchRequest) ([]byte, error) {
		t.Fatal("handler should not run for unauthorized origin")
		return nil, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Origin:     "https://evil.example",
		Capability: "dialog",
		Method:     "openFile",
	})
	assertDispatchCode(t, err, DispatchErrorUnauthorized)
}

func TestDispatcherRecoversHandlerPanic(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("app", "version", func(context.Context, DispatchRequest) ([]byte, error) {
		panic("boom")
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	resp, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Capability: "app",
		Method:     "version",
	})
	assertDispatchCode(t, err, DispatchErrorHandlerPanic)
	if resp.Error == nil || !strings.Contains(resp.Error.Detail, "goroutine") {
		t.Fatal("panic response did not include a stack detail")
	}
}

func TestDispatcherTimeoutCancelsHandler(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{DefaultTimeout: 5 * time.Millisecond})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("slow", "call", func(ctx context.Context, _ DispatchRequest) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Capability: "slow",
		Method:     "call",
	})
	assertDispatchCode(t, err, DispatchErrorTimeout)
}

func TestDispatcherTryDispatchBackpressure(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{QueueSize: 1})
	defer mustShutdownDispatcher(t, dispatcher)

	started := make(chan struct{})
	release := make(chan struct{})
	if err := dispatcher.Register("slow", "call", func(context.Context, DispatchRequest) ([]byte, error) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
		return []byte("ok"), nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	first, err := dispatcher.TryDispatch(context.Background(), DispatchRequest{Capability: "slow", Method: "call"})
	if err != nil {
		t.Fatalf("first TryDispatch() error = %v", err)
	}
	<-started
	second, err := dispatcher.TryDispatch(context.Background(), DispatchRequest{Capability: "slow", Method: "call"})
	if err != nil {
		t.Fatalf("second TryDispatch() error = %v", err)
	}
	_, err = dispatcher.TryDispatch(context.Background(), DispatchRequest{Capability: "slow", Method: "call"})
	assertDispatchCode(t, err, DispatchErrorBackpressure)

	close(release)
	for _, ch := range []<-chan DispatchResponse{first, second} {
		resp := <-ch
		if resp.Error != nil {
			t.Fatalf("queued response error = %v", resp.Error)
		}
	}
}

func TestDispatcherShutdownRejectsNewWork(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	if err := dispatcher.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	err := dispatcher.Register("app", "version", func(context.Context, DispatchRequest) ([]byte, error) {
		return nil, nil
	})
	assertDispatchCode(t, err, DispatchErrorClosed)

	_, err = dispatcher.Invoke(context.Background(), DispatchRequest{Capability: "app", Method: "version"})
	assertDispatchCode(t, err, DispatchErrorClosed)
}

func TestDispatcherHandlerErrorIsStructured(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	defer mustShutdownDispatcher(t, dispatcher)

	if err := dispatcher.Register("files", "readText", func(context.Context, DispatchRequest) ([]byte, error) {
		return nil, errors.New("disk said no")
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{
		Capability: "files",
		Method:     "readText",
	})
	assertDispatchCode(t, err, DispatchErrorHandlerFailed)
}

func TestDispatcherUnregister(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{})
	defer mustShutdownDispatcher(t, dispatcher)

	handler := func(context.Context, DispatchRequest) ([]byte, error) {
		return []byte("ok"), nil
	}
	if err := dispatcher.Register("app", "version", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := dispatcher.Unregister("app", "version"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}
	_, err := dispatcher.Invoke(context.Background(), DispatchRequest{Capability: "app", Method: "version"})
	assertDispatchCode(t, err, DispatchErrorHandlerNotFound)
}

func TestDispatcherBorrowPayloadOption(t *testing.T) {
	dispatcher := NewDispatcher(DispatcherOptions{BorrowPayload: true})
	defer mustShutdownDispatcher(t, dispatcher)

	gate := make(chan struct{})
	if err := dispatcher.Register("echo", "bytes", func(_ context.Context, req DispatchRequest) ([]byte, error) {
		<-gate
		return req.Payload, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	payload := []byte("abc")
	result, err := dispatcher.Dispatch(context.Background(), DispatchRequest{
		Capability: "echo",
		Method:     "bytes",
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}
	payload[0] = 'z'
	close(gate)

	resp := <-result
	if resp.Error != nil {
		t.Fatalf("Dispatch response error = %v", resp.Error)
	}
	if !bytes.Equal(resp.Payload, []byte("zbc")) {
		t.Fatalf("Payload = %q, want borrowed payload zbc", resp.Payload)
	}
}

func assertDispatchCode(t *testing.T, err error, code DispatchErrorCode) {
	t.Helper()
	var dispatchErr *DispatchError
	if !errors.As(err, &dispatchErr) {
		t.Fatalf("error = %v, want DispatchError %s", err, code)
	}
	if dispatchErr.Code != code {
		t.Fatalf("DispatchError code = %s, want %s", dispatchErr.Code, code)
	}
}

func mustShutdownDispatcher(t *testing.T, dispatcher *Dispatcher) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := dispatcher.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}
