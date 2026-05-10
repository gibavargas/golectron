package browserwindow

import (
	"errors"
	"fmt"
)

type Event string

const (
	EventShow       Event = "show"
	EventHide       Event = "hide"
	EventResize     Event = "resize"
	EventMove       Event = "move"
	EventMinimize   Event = "minimize"
	EventRestore    Event = "restore"
	EventMaximize   Event = "maximize"
	EventUnmaximize Event = "unmaximize"
	EventEnterFull  Event = "enter-full-screen"
	EventLeaveFull  Event = "leave-full-screen"
	EventClose      Event = "close"
	EventClosed     Event = "closed"
)

var ErrWindowClosed = errors.New("browser window is closed")

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type EventContext struct {
	Event     Event
	cancelled bool
}

func (ctx *EventContext) PreventDefault() {
	ctx.cancelled = true
}

func (ctx EventContext) DefaultPrevented() bool {
	return ctx.cancelled
}

type Handler func(*EventContext)

type Window struct {
	id         int64
	bounds     Rect
	normal     Rect
	visible    bool
	minimized  bool
	maximized  bool
	fullscreen bool
	closed     bool
	enabled    bool
	title      string
	handlers   map[Event][]Handler
	events     []Event
}

func NewWindow(id int64, opts NormalizedOptions) *Window {
	x, y := 0, 0
	if opts.X != nil {
		x = *opts.X
	}
	if opts.Y != nil {
		y = *opts.Y
	}
	bounds := Rect{X: x, Y: y, Width: opts.Width, Height: opts.Height}
	return &Window{
		id:       id,
		bounds:   bounds,
		normal:   bounds,
		visible:  opts.Show,
		enabled:  true,
		title:    opts.Title,
		handlers: make(map[Event][]Handler),
	}
}

func (w *Window) On(event Event, handler Handler) {
	if handler == nil {
		return
	}
	w.handlers[event] = append(w.handlers[event], handler)
}

func (w *Window) ID() int64 {
	return w.id
}

func (w *Window) Bounds() Rect {
	return w.bounds
}

func (w *Window) NormalBounds() Rect {
	return w.normal
}

func (w *Window) Events() []Event {
	return append([]Event(nil), w.events...)
}

func (w *Window) SetBounds(bounds Rect) error {
	if err := validateRect(bounds); err != nil {
		return err
	}
	if err := w.ensureOpen(); err != nil {
		return err
	}
	moved := bounds.X != w.bounds.X || bounds.Y != w.bounds.Y
	resized := bounds.Width != w.bounds.Width || bounds.Height != w.bounds.Height
	w.bounds = bounds
	if !w.maximized && !w.fullscreen {
		w.normal = bounds
	}
	if moved {
		w.emit(EventMove)
	}
	if resized {
		w.emit(EventResize)
	}
	return nil
}

func (w *Window) SetSize(width, height int) error {
	bounds := w.bounds
	bounds.Width = width
	bounds.Height = height
	return w.SetBounds(bounds)
}

func (w *Window) Show() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if !w.visible {
		w.visible = true
		w.emit(EventShow)
	}
	return nil
}

func (w *Window) Hide() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if w.visible {
		w.visible = false
		w.emit(EventHide)
	}
	return nil
}

func (w *Window) IsVisible() bool {
	return w.visible && !w.closed
}

func (w *Window) Title() string {
	return w.title
}

func (w *Window) Minimize() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if !w.minimized {
		w.minimized = true
		w.emit(EventMinimize)
	}
	return nil
}

func (w *Window) Restore() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if w.minimized {
		w.minimized = false
		w.emit(EventRestore)
	}
	return nil
}

func (w *Window) Maximize() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if !w.maximized {
		w.maximized = true
		w.emit(EventMaximize)
	}
	return nil
}

func (w *Window) Unmaximize() error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if w.maximized {
		w.maximized = false
		w.bounds = w.normal
		w.emit(EventUnmaximize)
	}
	return nil
}

func (w *Window) SetFullScreen(fullscreen bool) error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	if w.fullscreen == fullscreen {
		return nil
	}
	w.fullscreen = fullscreen
	if fullscreen {
		w.emit(EventEnterFull)
	} else {
		w.bounds = w.normal
		w.emit(EventLeaveFull)
	}
	return nil
}

func (w *Window) IsMinimized() bool {
	return w.minimized
}

func (w *Window) IsMaximized() bool {
	return w.maximized
}

func (w *Window) IsFullScreen() bool {
	return w.fullscreen
}

func (w *Window) SetEnabled(enabled bool) error {
	if err := w.ensureOpen(); err != nil {
		return err
	}
	w.enabled = enabled
	return nil
}

func (w *Window) IsEnabled() bool {
	return w.enabled && !w.closed
}

func (w *Window) Close() bool {
	if w.closed {
		return false
	}
	ctx := w.emit(EventClose)
	if ctx.DefaultPrevented() {
		return false
	}
	w.closed = true
	w.visible = false
	w.emit(EventClosed)
	return true
}

func (w *Window) IsClosed() bool {
	return w.closed
}

func (w *Window) ensureOpen() error {
	if w.closed {
		return ErrWindowClosed
	}
	return nil
}

func (w *Window) emit(event Event) EventContext {
	w.events = append(w.events, event)
	ctx := EventContext{Event: event}
	for _, handler := range w.handlers[event] {
		handler(&ctx)
	}
	return ctx
}

func validateRect(bounds Rect) error {
	if bounds.Width <= 0 {
		return fmt.Errorf("%w: width must be positive", ErrInvalidBounds)
	}
	if bounds.Height <= 0 {
		return fmt.Errorf("%w: height must be positive", ErrInvalidBounds)
	}
	return nil
}
