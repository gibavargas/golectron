package ipc

import (
	"context"
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
	}
}
