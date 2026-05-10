package session

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const PersistPrefix = "persist:"

var (
	ErrInvalidPartition       = errors.New("invalid session partition")
	ErrInvalidCookie          = errors.New("invalid cookie")
	ErrPermissionDenied       = errors.New("permission denied")
	ErrUnsupportedClearOption = errors.New("unsupported clear storage option")
)

type Registry struct {
	mu       sync.Mutex
	defaults *Session
	byName   map[string]*Session
}

func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]*Session)}
}

func (r *Registry) DefaultSession() *Session {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.defaults == nil {
		r.defaults = newSession("", true)
	}
	return r.defaults
}

func (r *Registry) FromPartition(partition string) (*Session, error) {
	partition = strings.TrimSpace(partition)
	if strings.ContainsAny(partition, "\x00\r\n") {
		return nil, fmt.Errorf("%w: control characters are not allowed", ErrInvalidPartition)
	}
	if partition == "" {
		return r.DefaultSession(), nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if existing := r.byName[partition]; existing != nil {
		return existing, nil
	}
	s := newSession(partition, strings.HasPrefix(partition, PersistPrefix))
	r.byName[partition] = s
	return s, nil
}

type PermissionRequest struct {
	RequestingURL string
	Permission    string
	Details       map[string]string
}

type PermissionHandler func(PermissionRequest) bool

type CookieChangeCause string

const (
	CookieChangeExplicit  CookieChangeCause = "explicit"
	CookieChangeOverwrite CookieChangeCause = "overwrite"
	CookieChangeExpired   CookieChangeCause = "expired"
)

type Cookie struct {
	URL      string
	Name     string
	Value    string
	Domain   string
	Path     string
	Secure   bool
	HTTPOnly bool
	Expires  time.Time
}

type CookieChange struct {
	Cookie  Cookie
	Removed bool
	Cause   CookieChangeCause
}

type ClearStorageOptions struct {
	Storages []string
	Origins  []string
	Quotas   []string
}

type Session struct {
	partition         string
	persistent        bool
	permissionHandler PermissionHandler
	cookies           map[string]Cookie
	cookieChanges     []CookieChange
	cacheCleared      bool
	storageClears     []ClearStorageOptions
	webauthnHandler   WebAuthnAccountSelectionHandler
}

func newSession(partition string, persistent bool) *Session {
	return &Session{
		partition:  partition,
		persistent: persistent,
		cookies:    make(map[string]Cookie),
	}
}

func (s *Session) Partition() string {
	return s.partition
}

func (s *Session) IsPersistent() bool {
	return s.persistent
}

func (s *Session) SetPermissionRequestHandler(handler PermissionHandler) {
	s.permissionHandler = handler
}

func (s *Session) CheckPermission(req PermissionRequest) bool {
	if s.permissionHandler == nil {
		return false
	}
	return s.permissionHandler(req)
}

func (s *Session) RequestPermission(req PermissionRequest) error {
	if !s.CheckPermission(req) {
		return fmt.Errorf("%w: %s", ErrPermissionDenied, req.Permission)
	}
	return nil
}

func (s *Session) SetCookie(cookie Cookie) error {
	normalized, err := normalizeCookie(cookie)
	if err != nil {
		return err
	}
	key := cookieKey(normalized)
	if existing, ok := s.cookies[key]; ok {
		s.cookieChanges = append(s.cookieChanges, CookieChange{Cookie: existing, Removed: true, Cause: CookieChangeOverwrite})
	}
	s.cookies[key] = normalized
	s.cookieChanges = append(s.cookieChanges, CookieChange{Cookie: normalized, Cause: CookieChangeExplicit})
	return nil
}

func (s *Session) Cookies() []Cookie {
	out := make([]Cookie, 0, len(s.cookies))
	for _, cookie := range s.cookies {
		out = append(out, cookie)
	}
	return out
}

func (s *Session) DeleteCookie(targetURL, name string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil || parsed.Hostname() == "" || strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: delete requires URL and name", ErrInvalidCookie)
	}
	host := strings.ToLower(parsed.Hostname())
	for key, cookie := range s.cookies {
		if cookie.Name == name && strings.EqualFold(strings.TrimPrefix(cookie.Domain, "."), host) {
			delete(s.cookies, key)
			s.cookieChanges = append(s.cookieChanges, CookieChange{Cookie: cookie, Removed: true, Cause: CookieChangeExplicit})
		}
	}
	return nil
}

func (s *Session) CookieChanges() []CookieChange {
	return append([]CookieChange(nil), s.cookieChanges...)
}

func (s *Session) ClearCache() {
	s.cacheCleared = true
}

func (s *Session) CacheCleared() bool {
	return s.cacheCleared
}

func (s *Session) ClearStorageData(options ClearStorageOptions) error {
	originHosts := make(map[string]bool)
	for _, origin := range options.Origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("%w: invalid origin %q", ErrUnsupportedClearOption, origin)
		}
		originHosts[strings.ToLower(parsed.Hostname())] = true
	}
	if shouldClearStorage(options.Storages, "cookies") {
		for key, cookie := range s.cookies {
			host := strings.TrimPrefix(strings.ToLower(cookie.Domain), ".")
			if len(originHosts) == 0 || originHosts[host] {
				delete(s.cookies, key)
				s.cookieChanges = append(s.cookieChanges, CookieChange{Cookie: cookie, Removed: true, Cause: CookieChangeExplicit})
			}
		}
	}
	options.Quotas = nil
	s.storageClears = append(s.storageClears, cloneClearStorageOptions(options))
	return nil
}

func (s *Session) StorageClears() []ClearStorageOptions {
	out := make([]ClearStorageOptions, len(s.storageClears))
	for i, clear := range s.storageClears {
		out[i] = cloneClearStorageOptions(clear)
	}
	return out
}

func normalizeCookie(cookie Cookie) (Cookie, error) {
	if strings.TrimSpace(cookie.Name) == "" || strings.ContainsAny(cookie.Name, ";\r\n") {
		return Cookie{}, fmt.Errorf("%w: invalid name", ErrInvalidCookie)
	}
	parsed, err := url.Parse(cookie.URL)
	if err != nil || parsed.Hostname() == "" {
		return Cookie{}, fmt.Errorf("%w: URL with host is required", ErrInvalidCookie)
	}
	host := strings.ToLower(parsed.Hostname())
	if cookie.Domain == "" {
		cookie.Domain = host
	} else {
		cookie.Domain = strings.ToLower(strings.TrimSpace(cookie.Domain))
	}
	if cookie.Path == "" {
		cookie.Path = "/"
	}
	if cookie.Secure && parsed.Scheme != "https" {
		return Cookie{}, fmt.Errorf("%w: secure cookies require https", ErrInvalidCookie)
	}
	cookie.URL = parsed.String()
	return cookie, nil
}

func cookieKey(cookie Cookie) string {
	return strings.ToLower(cookie.Domain) + "\x00" + cookie.Path + "\x00" + cookie.Name
}

func cloneClearStorageOptions(options ClearStorageOptions) ClearStorageOptions {
	return ClearStorageOptions{
		Storages: append([]string(nil), options.Storages...),
		Origins:  append([]string(nil), options.Origins...),
		Quotas:   append([]string(nil), options.Quotas...),
	}
}

func shouldClearStorage(storages []string, target string) bool {
	if len(storages) == 0 {
		return true
	}
	for _, storage := range storages {
		if strings.EqualFold(strings.TrimSpace(storage), target) {
			return true
		}
	}
	return false
}
