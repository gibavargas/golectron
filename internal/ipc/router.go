package ipc

import (
	"context"
	"fmt"
	"sync"
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

type Router struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]Handler)}
}

func (r *Router) Register(channel string, handler Handler) error {
	if channel == "" {
		return fmt.Errorf("channel is required")
	}
	if handler == nil {
		return fmt.Errorf("handler is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[channel]; exists {
		return fmt.Errorf("channel already registered: %s", channel)
	}
	r.handlers[channel] = handler
	return nil
}

func (r *Router) Invoke(ctx context.Context, msg Message) (Reply, error) {
	if msg.Channel == "" {
		return Reply{}, fmt.Errorf("channel is required")
	}

	r.mu.RLock()
	handler, ok := r.handlers[msg.Channel]
	r.mu.RUnlock()
	if !ok {
		return Reply{}, fmt.Errorf("channel not registered: %s", msg.Channel)
	}

	return handler(ctx, msg)
}
