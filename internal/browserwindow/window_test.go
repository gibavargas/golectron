package browserwindow

import (
	"errors"
	"reflect"
	"testing"
)

func TestWindowBoundsSizeAndNormalBounds(t *testing.T) {
	opts, err := NormalizeOptions(ConstructorOptions{})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	win := NewWindow(7, opts)
	if win.ID() != 7 {
		t.Fatalf("ID() = %d, want 7", win.ID())
	}
	if got := win.Bounds(); got.Width != DefaultWidth || got.Height != DefaultHeight {
		t.Fatalf("Bounds() = %#v, want defaults", got)
	}
	if err := win.SetBounds(Rect{X: 10, Y: 20, Width: 1024, Height: 768}); err != nil {
		t.Fatalf("SetBounds() error = %v", err)
	}
	if got := win.NormalBounds(); got != (Rect{X: 10, Y: 20, Width: 1024, Height: 768}) {
		t.Fatalf("NormalBounds() = %#v", got)
	}
	wantEvents := []Event{EventMove, EventResize}
	if got := win.Events(); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("Events() = %#v, want %#v", got, wantEvents)
	}
	if err := win.SetSize(640, 480); err != nil {
		t.Fatalf("SetSize() error = %v", err)
	}
	if got := win.Bounds(); got.Width != 640 || got.Height != 480 {
		t.Fatalf("Bounds() after SetSize = %#v", got)
	}
}

func TestWindowVisibilityAndStateEvents(t *testing.T) {
	show := false
	opts, _ := NormalizeOptions(ConstructorOptions{Show: &show})
	win := NewWindow(1, opts)
	if win.IsVisible() {
		t.Fatal("IsVisible() = true for show:false")
	}
	win.Show()
	win.Hide()
	win.Minimize()
	win.Restore()
	win.Maximize()
	win.Unmaximize()
	win.SetFullScreen(true)
	win.SetFullScreen(false)

	want := []Event{
		EventShow,
		EventHide,
		EventMinimize,
		EventRestore,
		EventMaximize,
		EventUnmaximize,
		EventEnterFull,
		EventLeaveFull,
	}
	if got := win.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if win.IsMinimized() || win.IsMaximized() || win.IsFullScreen() {
		t.Fatalf("window state not restored: minimized=%v maximized=%v fullscreen=%v", win.IsMinimized(), win.IsMaximized(), win.IsFullScreen())
	}
}

func TestWindowCloseCanBeCancelled(t *testing.T) {
	opts, _ := NormalizeOptions(ConstructorOptions{})
	win := NewWindow(1, opts)
	win.On(EventClose, func(ctx *EventContext) {
		ctx.PreventDefault()
	})
	if win.Close() {
		t.Fatal("Close() = true after preventDefault")
	}
	if win.IsClosed() {
		t.Fatal("IsClosed() = true after cancelled close")
	}
	if err := win.SetSize(100, 100); err != nil {
		t.Fatalf("SetSize() after cancelled close error = %v", err)
	}
}

func TestWindowCloseEmitsClosedAndRejectsMutation(t *testing.T) {
	opts, _ := NormalizeOptions(ConstructorOptions{})
	win := NewWindow(1, opts)
	var closeSeen bool
	win.On(EventClose, func(ctx *EventContext) {
		closeSeen = true
	})
	if !win.Close() {
		t.Fatal("Close() = false, want true")
	}
	if !closeSeen || !win.IsClosed() || win.IsVisible() || win.IsEnabled() {
		t.Fatalf("closed state invalid: closeSeen=%v closed=%v visible=%v enabled=%v", closeSeen, win.IsClosed(), win.IsVisible(), win.IsEnabled())
	}
	want := []Event{EventClose, EventClosed}
	if got := win.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if err := win.SetSize(100, 100); !errors.Is(err, ErrWindowClosed) {
		t.Fatalf("SetSize(closed) error = %v, want ErrWindowClosed", err)
	}
	if win.Close() {
		t.Fatal("second Close() = true, want false")
	}
}

func TestWindowValidation(t *testing.T) {
	opts, _ := NormalizeOptions(ConstructorOptions{})
	win := NewWindow(1, opts)
	if err := win.SetBounds(Rect{Width: 0, Height: 1}); !errors.Is(err, ErrInvalidBounds) {
		t.Fatalf("SetBounds(width=0) error = %v, want ErrInvalidBounds", err)
	}
	if err := win.SetBounds(Rect{Width: 1, Height: -1}); !errors.Is(err, ErrInvalidBounds) {
		t.Fatalf("SetBounds(height=-1) error = %v, want ErrInvalidBounds", err)
	}
}
