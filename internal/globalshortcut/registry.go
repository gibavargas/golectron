package globalshortcut

import (
	"errors"
	"strings"
	"sync"
)

var (
	ErrAcceleratorRequired = errors.New("accelerator is required")
	ErrInvalidAccelerator  = errors.New("invalid accelerator")
	ErrCallbackRequired    = errors.New("callback is required")
)

type Callback func()

type Registry struct {
	mu          sync.RWMutex
	callbacks   map[string]Callback
	suspended   bool
	triggerHook func(string)
}

func NewRegistry() *Registry {
	return &Registry{callbacks: make(map[string]Callback)}
}

func (r *Registry) Register(accelerator string, callback Callback) (bool, error) {
	parsed, err := ParseAccelerator(accelerator)
	if err != nil {
		return false, err
	}
	if callback == nil {
		return false, ErrCallbackRequired
	}
	canonical := parsed.String()

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.callbacks[canonical]; exists {
		return false, nil
	}
	r.callbacks[canonical] = callback
	return true, nil
}

func (r *Registry) RegisterAll(accelerators []string, callback Callback) map[string]bool {
	results := make(map[string]bool, len(accelerators))
	for _, accelerator := range accelerators {
		registered, err := r.Register(accelerator, callback)
		canonical := normalizeAccelerator(accelerator)
		if parsed, parseErr := ParseAccelerator(accelerator); parseErr == nil {
			canonical = parsed.String()
		}
		if err != nil {
			results[canonical] = false
			continue
		}
		results[canonical] = registered
	}
	return results
}

func (r *Registry) IsRegistered(accelerator string) bool {
	canonical, ok := canonicalAccelerator(accelerator)
	if !ok {
		return false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok = r.callbacks[canonical]
	return ok
}

func (r *Registry) Unregister(accelerator string) {
	canonical, ok := canonicalAccelerator(accelerator)
	if !ok {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.callbacks, canonical)
}

func (r *Registry) UnregisterAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.callbacks = make(map[string]Callback)
}

func (r *Registry) SetSuspended(suspended bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.suspended = suspended
}

func (r *Registry) IsSuspended() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.suspended
}

func (r *Registry) Trigger(accelerator string) bool {
	canonical, ok := canonicalAccelerator(accelerator)
	if !ok {
		return false
	}
	r.mu.RLock()
	callback, ok := r.callbacks[canonical]
	suspended := r.suspended
	hook := r.triggerHook
	r.mu.RUnlock()
	if !ok {
		return false
	}
	if suspended {
		return true
	}
	if hook != nil {
		hook(canonical)
	}
	callback()
	return true
}

func (r *Registry) SetTriggerHook(hook func(string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.triggerHook = hook
}

func normalizeAccelerator(accelerator string) string {
	return strings.TrimSpace(accelerator)
}

func canonicalAccelerator(accelerator string) (string, bool) {
	parsed, err := ParseAccelerator(accelerator)
	if err != nil {
		return "", false
	}
	return parsed.String(), true
}
