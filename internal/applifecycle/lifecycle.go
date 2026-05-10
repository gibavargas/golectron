package applifecycle

import (
	"errors"
	"sync"
)

type Event string

const (
	EventWillFinishLaunching Event = "will-finish-launching"
	EventReady               Event = "ready"
	EventWindowAllClosed     Event = "window-all-closed"
	EventBeforeQuit          Event = "before-quit"
	EventWillQuit            Event = "will-quit"
	EventQuit                Event = "quit"
)

var ErrAlreadyReady = errors.New("app lifecycle is already ready")

type EventContext struct {
	Event     Event
	ExitCode  int
	cancelled bool
}

func (ctx *EventContext) PreventDefault() {
	ctx.cancelled = true
}

func (ctx EventContext) DefaultPrevented() bool {
	return ctx.cancelled
}

type Handler func(*EventContext)

type App struct {
	mu            sync.Mutex
	handlers      map[Event][]Handler
	ready         bool
	quitting      bool
	quitExitCode  int
	readyWaiters  []func()
	emittedEvents []Event
}

func New() *App {
	return &App{handlers: make(map[Event][]Handler)}
}

func (app *App) On(event Event, handler Handler) {
	if handler == nil {
		return
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.handlers[event] = append(app.handlers[event], handler)
}

func (app *App) IsReady() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.ready
}

func (app *App) WhenReady(handler func()) {
	if handler == nil {
		return
	}
	app.mu.Lock()
	ready := app.ready
	if !ready {
		app.readyWaiters = append(app.readyWaiters, handler)
	}
	app.mu.Unlock()
	if ready {
		handler()
	}
}

func (app *App) MarkReady() error {
	app.mu.Lock()
	if app.ready {
		app.mu.Unlock()
		return ErrAlreadyReady
	}
	app.ready = true
	waiters := append([]func(){}, app.readyWaiters...)
	app.readyWaiters = nil
	app.mu.Unlock()

	app.Emit(EventWillFinishLaunching, 0)
	app.Emit(EventReady, 0)
	for _, waiter := range waiters {
		waiter()
	}
	return nil
}

func (app *App) WindowAllClosed() bool {
	app.Emit(EventWindowAllClosed, 0)
	app.mu.Lock()
	hasHandler := len(app.handlers[EventWindowAllClosed]) > 0
	app.mu.Unlock()
	if hasHandler {
		return false
	}
	return app.Quit(0)
}

func (app *App) Quit(exitCode int) bool {
	if cancelled := app.emitCancellable(EventBeforeQuit, exitCode); cancelled {
		return false
	}
	if cancelled := app.emitCancellable(EventWillQuit, exitCode); cancelled {
		return false
	}

	app.mu.Lock()
	app.quitting = true
	app.quitExitCode = exitCode
	app.mu.Unlock()

	app.Emit(EventQuit, exitCode)
	return true
}

func (app *App) Exit(exitCode int) {
	app.mu.Lock()
	app.quitting = true
	app.quitExitCode = exitCode
	app.mu.Unlock()
	app.Emit(EventQuit, exitCode)
}

func (app *App) IsQuitting() bool {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.quitting
}

func (app *App) ExitCode() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.quitExitCode
}

func (app *App) Events() []Event {
	app.mu.Lock()
	defer app.mu.Unlock()
	return append([]Event(nil), app.emittedEvents...)
}

func (app *App) Emit(event Event, exitCode int) EventContext {
	app.mu.Lock()
	handlers := append([]Handler(nil), app.handlers[event]...)
	app.emittedEvents = append(app.emittedEvents, event)
	app.mu.Unlock()

	ctx := EventContext{Event: event, ExitCode: exitCode}
	for _, handler := range handlers {
		handler(&ctx)
	}
	return ctx
}

func (app *App) emitCancellable(event Event, exitCode int) bool {
	ctx := app.Emit(event, exitCode)
	return ctx.DefaultPrevented()
}
