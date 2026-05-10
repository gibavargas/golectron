package protocol

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

var (
	ErrInvalidScheme     = errors.New("invalid protocol scheme")
	ErrInvalidPrivileges = errors.New("invalid protocol privileges")
	ErrAlreadyRegistered = errors.New("protocol already registered")
	ErrNotRegistered     = errors.New("protocol not registered")
	ErrRegistryLocked    = errors.New("protocol privilege registry is locked")
	ErrInvalidResponse   = errors.New("invalid protocol response")
)

var schemePattern = regexp.MustCompile(`^[a-z][a-z0-9+.-]*$`)

type Privileges struct {
	Standard            bool
	Secure              bool
	BypassCSP           bool
	SupportFetchAPI     bool
	CorsEnabled         bool
	Stream              bool
	AllowServiceWorkers bool
	AllowExtensions     bool
}

type SchemePrivilege struct {
	Scheme     string
	Privileges Privileges
}

type HandlerKind string

const (
	HandlerString HandlerKind = "string"
	HandlerBuffer HandlerKind = "buffer"
	HandlerFile   HandlerKind = "file"
	HandlerHTTP   HandlerKind = "http"
	HandlerStream HandlerKind = "stream"
)

type Handler struct {
	Kind HandlerKind
}

type Response struct {
	StatusCode int
	Headers    http.Header
}

type Registry struct {
	locked     bool
	privileges map[string]Privileges
	handlers   map[string]Handler
}

func NewRegistry() *Registry {
	return &Registry{
		privileges: make(map[string]Privileges),
		handlers:   make(map[string]Handler),
	}
}

func (r *Registry) RegisterSchemesAsPrivileged(schemes []SchemePrivilege) error {
	if r.locked {
		return ErrRegistryLocked
	}
	for _, scheme := range schemes {
		name, err := NormalizeScheme(scheme.Scheme)
		if err != nil {
			return err
		}
		if _, exists := r.privileges[name]; exists {
			return fmt.Errorf("%w: %s", ErrAlreadyRegistered, name)
		}
		if scheme.Privileges.AllowExtensions && !scheme.Privileges.Standard {
			return fmt.Errorf("%w: allowExtensions requires a standard scheme", ErrInvalidPrivileges)
		}
		r.privileges[name] = scheme.Privileges
	}
	return nil
}

func (r *Registry) LockPrivileges() {
	r.locked = true
}

func (r *Registry) PrivilegesFor(scheme string) (Privileges, bool) {
	name, err := NormalizeScheme(scheme)
	if err != nil {
		return Privileges{}, false
	}
	privileges, ok := r.privileges[name]
	return privileges, ok
}

func (r *Registry) RegisterHandler(scheme string, handler Handler) error {
	name, err := NormalizeScheme(scheme)
	if err != nil {
		return err
	}
	if handler.Kind == "" {
		return fmt.Errorf("%w: handler kind is required", ErrNotRegistered)
	}
	if _, exists := r.handlers[name]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyRegistered, name)
	}
	r.handlers[name] = handler
	return nil
}

func (r *Registry) IsProtocolHandled(scheme string) bool {
	name, err := NormalizeScheme(scheme)
	if err != nil {
		return false
	}
	_, ok := r.handlers[name]
	return ok
}

func (r *Registry) UnregisterProtocol(scheme string) error {
	name, err := NormalizeScheme(scheme)
	if err != nil {
		return err
	}
	if _, exists := r.handlers[name]; !exists {
		return fmt.Errorf("%w: %s", ErrNotRegistered, name)
	}
	delete(r.handlers, name)
	return nil
}

func NormalizeScheme(scheme string) (string, error) {
	normalized := strings.TrimSpace(scheme)
	normalized = strings.TrimSuffix(normalized, ":")
	if normalized == "" || normalized != strings.ToLower(normalized) || !schemePattern.MatchString(normalized) {
		return "", fmt.Errorf("%w: %q", ErrInvalidScheme, scheme)
	}
	return normalized, nil
}

func ValidateResponse(response Response) error {
	if response.StatusCode < 100 || response.StatusCode > 599 {
		return fmt.Errorf("%w: status code", ErrInvalidResponse)
	}
	for name, values := range response.Headers {
		if strings.TrimSpace(name) == "" || strings.ContainsAny(name, ":\r\n") {
			return fmt.Errorf("%w: header name", ErrInvalidResponse)
		}
		for _, value := range values {
			if strings.ContainsAny(value, "\r\n") {
				return fmt.Errorf("%w: header value", ErrInvalidResponse)
			}
		}
	}
	return nil
}
