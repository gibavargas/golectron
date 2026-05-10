package ipc

import (
	"context"
	"errors"
	"testing"
)

func TestRouterInvoke(t *testing.T) {
	router := NewRouter()
	if err := router.Register("ping", func(context.Context, Message) (Reply, error) {
		return Reply{Value: "pong"}, nil
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	reply, err := router.Invoke(context.Background(), Message{Channel: "ping"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if reply.Value != "pong" {
		t.Fatalf("reply.Value = %v, want pong", reply.Value)
	}
}

func TestRouterRejectsDuplicateChannel(t *testing.T) {
	router := NewRouter()
	handler := func(context.Context, Message) (Reply, error) { return Reply{}, nil }

	if err := router.Register("dupe", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := router.Register("dupe", handler); err == nil {
		t.Fatal("Register() duplicate error = nil, want error")
	} else if !errors.Is(err, ErrHandlerExists) {
		t.Fatalf("Register() duplicate error = %v, want ErrHandlerExists", err)
	}
}

func TestRouterHandleOnceRemovesBeforeInvoke(t *testing.T) {
	router := NewRouter()
	calls := 0
	if err := router.HandleOnce("once", func(context.Context, Message) (Reply, error) {
		calls++
		if router.HasHandler("once") {
			t.Fatal("HandleOnce handler still registered during invocation")
		}
		return Reply{Value: calls}, nil
	}); err != nil {
		t.Fatalf("HandleOnce() error = %v", err)
	}

	reply, err := router.Invoke(context.Background(), Message{Channel: "once"})
	if err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if reply.Value != 1 {
		t.Fatalf("reply.Value = %v, want 1", reply.Value)
	}
	if _, err := router.Invoke(context.Background(), Message{Channel: "once"}); !errors.Is(err, ErrNoHandler) {
		t.Fatalf("second Invoke() error = %v, want ErrNoHandler", err)
	}
}

func TestRouterRemoveHandler(t *testing.T) {
	router := NewRouter()
	if err := router.Handle("gone", func(context.Context, Message) (Reply, error) {
		return Reply{Value: "unreachable"}, nil
	}); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}
	if !router.HasHandler("gone") {
		t.Fatal("HasHandler() = false, want true")
	}
	router.RemoveHandler("gone")
	if router.HasHandler("gone") {
		t.Fatal("HasHandler() = true after RemoveHandler")
	}
	if _, err := router.Invoke(context.Background(), Message{Channel: "gone"}); !errors.Is(err, ErrNoHandler) {
		t.Fatalf("Invoke() error = %v, want ErrNoHandler", err)
	}
}

func TestRouterRejectsInvalidHandlers(t *testing.T) {
	router := NewRouter()
	if err := router.Handle("", func(context.Context, Message) (Reply, error) {
		return Reply{}, nil
	}); !errors.Is(err, ErrChannelRequired) {
		t.Fatalf("Handle(empty channel) error = %v, want ErrChannelRequired", err)
	}
	if err := router.Handle("missing", nil); !errors.Is(err, ErrHandlerRequired) {
		t.Fatalf("Handle(nil handler) error = %v, want ErrHandlerRequired", err)
	}
	if _, err := router.Invoke(context.Background(), Message{}); !errors.Is(err, ErrChannelRequired) {
		t.Fatalf("Invoke(empty channel) error = %v, want ErrChannelRequired", err)
	}
}
