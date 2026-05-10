package webcontents

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
)

type Event string

const (
	EventDidStartNavigation Event = "did-start-navigation"
	EventWillFrameNavigate  Event = "will-frame-navigate"
	EventWillNavigate       Event = "will-navigate"
	EventDidFrameNavigate   Event = "did-frame-navigate"
	EventDidNavigate        Event = "did-navigate"
	EventDidNavigateInPage  Event = "did-navigate-in-page"
	EventDidStartLoading    Event = "did-start-loading"
	EventDidStopLoading     Event = "did-stop-loading"
	EventDidFinishLoad      Event = "did-finish-load"
	EventDidFailLoad        Event = "did-fail-load"
)

var (
	ErrInvalidURL          = errors.New("invalid URL")
	ErrNavigationCancelled = errors.New("navigation cancelled")
	ErrHistoryUnavailable  = errors.New("history entry unavailable")
)

type NavigationDetails struct {
	URL            string
	IsSameDocument bool
	IsMainFrame    bool
	ErrorCode      int
	Error          string
}

type EventContext struct {
	Event     Event
	Details   NavigationDetails
	cancelled bool
}

func (ctx *EventContext) PreventDefault() {
	ctx.cancelled = true
}

func (ctx EventContext) DefaultPrevented() bool {
	return ctx.cancelled
}

type Handler func(*EventContext)

type WebContents struct {
	id        int64
	current   string
	history   []string
	index     int
	loading   bool
	targetID  string
	lastPrint *PrintJob
	handlers  map[Event][]Handler
	events    []Event
}

func New(id int64) *WebContents {
	return &WebContents{
		id:       id,
		index:    -1,
		handlers: make(map[Event][]Handler),
	}
}

func (wc *WebContents) ID() int64 {
	return wc.id
}

func (wc *WebContents) GetOrCreateDevToolsTargetID() string {
	if wc.targetID == "" {
		wc.targetID = fmt.Sprintf("electron-go-webcontents-%d", wc.id)
		registerDevToolsTarget(wc.targetID, wc)
	}
	return wc.targetID
}

func (wc *WebContents) On(event Event, handler Handler) {
	if handler == nil {
		return
	}
	wc.handlers[event] = append(wc.handlers[event], handler)
}

func (wc *WebContents) LoadURL(rawURL string) error {
	if err := validateURL(rawURL); err != nil {
		return err
	}
	details := NavigationDetails{URL: rawURL, IsMainFrame: true}
	wc.emit(EventDidStartNavigation, details)
	if wc.emit(EventWillFrameNavigate, details).DefaultPrevented() {
		return ErrNavigationCancelled
	}
	if wc.emit(EventWillNavigate, details).DefaultPrevented() {
		return ErrNavigationCancelled
	}
	wc.loading = true
	wc.emit(EventDidStartLoading, details)
	wc.commitNavigation(rawURL)
	wc.emit(EventDidFrameNavigate, details)
	wc.emit(EventDidNavigate, details)
	wc.loading = false
	wc.emit(EventDidStopLoading, details)
	wc.emit(EventDidFinishLoad, details)
	return nil
}

func (wc *WebContents) NavigateInPage(rawURL string) error {
	if err := validateURL(rawURL); err != nil {
		return err
	}
	details := NavigationDetails{URL: rawURL, IsSameDocument: true, IsMainFrame: true}
	wc.emit(EventDidStartNavigation, details)
	wc.commitNavigation(rawURL)
	wc.emit(EventDidNavigateInPage, details)
	return nil
}

func (wc *WebContents) FailLoad(rawURL string, code int, description string) error {
	if err := validateURL(rawURL); err != nil {
		return err
	}
	details := NavigationDetails{URL: rawURL, IsMainFrame: true, ErrorCode: code, Error: description}
	wc.loading = true
	wc.emit(EventDidStartLoading, details)
	wc.loading = false
	wc.emit(EventDidStopLoading, details)
	wc.emit(EventDidFailLoad, details)
	return nil
}

func (wc *WebContents) CanGoBack() bool {
	return wc.index > 0
}

func (wc *WebContents) CanGoForward() bool {
	return wc.index >= 0 && wc.index < len(wc.history)-1
}

func (wc *WebContents) GoBack() error {
	if !wc.CanGoBack() {
		return ErrHistoryUnavailable
	}
	wc.index--
	wc.current = wc.history[wc.index]
	wc.emit(EventDidNavigate, NavigationDetails{URL: wc.current, IsMainFrame: true})
	return nil
}

func (wc *WebContents) GoForward() error {
	if !wc.CanGoForward() {
		return ErrHistoryUnavailable
	}
	wc.index++
	wc.current = wc.history[wc.index]
	wc.emit(EventDidNavigate, NavigationDetails{URL: wc.current, IsMainFrame: true})
	return nil
}

func (wc *WebContents) URL() string {
	return wc.current
}

func (wc *WebContents) IsLoading() bool {
	return wc.loading
}

func (wc *WebContents) History() []string {
	return append([]string(nil), wc.history...)
}

func (wc *WebContents) Events() []Event {
	return append([]Event(nil), wc.events...)
}

func (wc *WebContents) commitNavigation(rawURL string) {
	if wc.index < len(wc.history)-1 {
		wc.history = append([]string(nil), wc.history[:wc.index+1]...)
	}
	wc.history = append(wc.history, rawURL)
	wc.index = len(wc.history) - 1
	wc.current = rawURL
}

func (wc *WebContents) emit(event Event, details NavigationDetails) EventContext {
	ctx := EventContext{Event: event, Details: details}
	wc.events = append(wc.events, event)
	for _, handler := range wc.handlers[event] {
		handler(&ctx)
	}
	return ctx
}

func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return fmt.Errorf("%w: %s", ErrInvalidURL, rawURL)
	}
	return nil
}

var devToolsTargets = struct {
	sync.RWMutex
	byID map[string]*WebContents
}{byID: make(map[string]*WebContents)}

func FromDevToolsTargetID(targetID string) (*WebContents, bool) {
	devToolsTargets.RLock()
	defer devToolsTargets.RUnlock()
	contents, ok := devToolsTargets.byID[targetID]
	return contents, ok
}

func registerDevToolsTarget(targetID string, contents *WebContents) {
	devToolsTargets.Lock()
	defer devToolsTargets.Unlock()
	devToolsTargets.byID[targetID] = contents
}
