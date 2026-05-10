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
	handlers map[string]handlerEntry
}

type handlerEntry struct {
	handler Handler
	once    bool
}

func NewRouter() *Router {
	return &Router{handlers: make(map[string]handlerEntry)}
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
