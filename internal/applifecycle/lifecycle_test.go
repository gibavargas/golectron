package applifecycle

import (
	"errors"
	"reflect"
	"testing"
)

func TestMarkReadyEmitsLaunchAndReadyOnce(t *testing.T) {
	app := New()
	waiterCalls := 0
	app.WhenReady(func() {
		waiterCalls++
	})

	if err := app.MarkReady(); err != nil {
		t.Fatalf("MarkReady() error = %v", err)
	}
	if !app.IsReady() {
		t.Fatal("IsReady() = false, want true")
	}
	if waiterCalls != 1 {
		t.Fatalf("waiterCalls = %d, want 1", waiterCalls)
	}
	app.WhenReady(func() {
		waiterCalls++
	})
	if waiterCalls != 2 {
		t.Fatalf("late waiterCalls = %d, want 2", waiterCalls)
	}
	if err := app.MarkReady(); !errors.Is(err, ErrAlreadyReady) {
		t.Fatalf("second MarkReady() error = %v, want ErrAlreadyReady", err)
	}

	wantEvents := []Event{EventWillFinishLaunching, EventReady}
	if got := app.Events(); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("Events() = %#v, want %#v", got, wantEvents)
	}
}

func TestQuitEmitsCancellableQuitSequence(t *testing.T) {
	app := New()
	var got []Event
	for _, event := range []Event{EventBeforeQuit, EventWillQuit, EventQuit} {
		event := event
		app.On(event, func(ctx *EventContext) {
			got = append(got, ctx.Event)
			if ctx.ExitCode != 7 {
				t.Fatalf("%s ExitCode = %d, want 7", ctx.Event, ctx.ExitCode)
			}
		})
	}

	if !app.Quit(7) {
		t.Fatal("Quit() = false, want true")
	}
	want := []Event{EventBeforeQuit, EventWillQuit, EventQuit}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("handler order = %#v, want %#v", got, want)
	}
	if !app.IsQuitting() {
		t.Fatal("IsQuitting() = false, want true")
	}
	if app.ExitCode() != 7 {
		t.Fatalf("ExitCode() = %d, want 7", app.ExitCode())
	}
}

func TestQuitCanBeCancelledBeforeOrWillQuit(t *testing.T) {
	tests := []Event{EventBeforeQuit, EventWillQuit}
	for _, cancelAt := range tests {
		t.Run(string(cancelAt), func(t *testing.T) {
			app := New()
			var got []Event
			for _, event := range []Event{EventBeforeQuit, EventWillQuit, EventQuit} {
				event := event
				app.On(event, func(ctx *EventContext) {
					got = append(got, ctx.Event)
					if ctx.Event == cancelAt {
						ctx.PreventDefault()
					}
				})
			}

			if app.Quit(0) {
				t.Fatal("Quit() = true after cancellation")
			}
			if app.IsQuitting() {
				t.Fatal("IsQuitting() = true after cancellation")
			}
			if len(got) == 0 || got[len(got)-1] != cancelAt {
				t.Fatalf("last event = %#v, want cancellation at %s", got, cancelAt)
			}
			for _, event := range got {
				if event == EventQuit {
					t.Fatalf("quit emitted after cancellation: %#v", got)
				}
			}
		})
	}
}

func TestExitSkipsBeforeAndWillQuit(t *testing.T) {
	app := New()
	app.On(EventBeforeQuit, func(*EventContext) {
		t.Fatal("before-quit emitted during Exit")
	})
	app.On(EventWillQuit, func(*EventContext) {
		t.Fatal("will-quit emitted during Exit")
	})

	quitSeen := false
	app.On(EventQuit, func(ctx *EventContext) {
		quitSeen = true
		if ctx.ExitCode != 42 {
			t.Fatalf("ExitCode = %d, want 42", ctx.ExitCode)
		}
	})
	app.Exit(42)
	if !quitSeen {
		t.Fatal("quit was not emitted")
	}
	if !app.IsQuitting() {
		t.Fatal("IsQuitting() = false, want true")
	}
}

func TestWindowAllClosedDefaultQuitAndSubscribedOverride(t *testing.T) {
	defaultApp := New()
	if !defaultApp.WindowAllClosed() {
		t.Fatal("WindowAllClosed() = false without subscriber, want default quit")
	}
	if !defaultApp.IsQuitting() {
		t.Fatal("default app did not quit after window-all-closed")
	}

	subscribedApp := New()
	seen := false
	subscribedApp.On(EventWindowAllClosed, func(*EventContext) {
		seen = true
	})
	if subscribedApp.WindowAllClosed() {
		t.Fatal("WindowAllClosed() = true with subscriber, want controlled non-quit")
	}
	if !seen {
		t.Fatal("window-all-closed subscriber was not called")
	}
	if subscribedApp.IsQuitting() {
		t.Fatal("subscribed app quit despite window-all-closed subscriber")
	}
}
