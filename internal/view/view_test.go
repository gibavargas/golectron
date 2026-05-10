package view

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestSetBoundsStoresAnimationAndEmitsBoundsChanged(t *testing.T) {
	v := New(1)
	var events []Event
	v.On(EventBoundsChanged, func(event Event) {
		events = append(events, event)
	})
	opts := SetBoundsOptions{Animate: true, Duration: 250 * time.Millisecond}
	if err := v.SetBounds(Rect{X: 1, Y: 2, Width: 300, Height: 200}, opts); err != nil {
		t.Fatalf("SetBounds() error = %v", err)
	}
	if got := v.Bounds(); got != (Rect{X: 1, Y: 2, Width: 300, Height: 200}) {
		t.Fatalf("Bounds() = %#v", got)
	}
	if got := v.LastAnimation(); got != opts {
		t.Fatalf("LastAnimation() = %#v, want %#v", got, opts)
	}
	if !reflect.DeepEqual(events, []Event{EventBoundsChanged}) {
		t.Fatalf("handler events = %#v", events)
	}
	if !reflect.DeepEqual(v.Events(), []Event{EventBoundsChanged}) {
		t.Fatalf("Events() = %#v", v.Events())
	}
}

func TestSetBoundsValidation(t *testing.T) {
	v := New(1)
	tests := []struct {
		name   string
		bounds Rect
		opts   SetBoundsOptions
	}{
		{name: "negative width", bounds: Rect{Width: -1, Height: 1}},
		{name: "negative height", bounds: Rect{Width: 1, Height: -1}},
		{name: "negative duration", bounds: Rect{Width: 1, Height: 1}, opts: SetBoundsOptions{Duration: -time.Second}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := v.SetBounds(tt.bounds, tt.opts); !errors.Is(err, ErrInvalidBounds) {
				t.Fatalf("SetBounds() error = %v, want ErrInvalidBounds", err)
			}
		})
	}
}

func TestBackgroundBlurColorBorderRadiusAndVisibility(t *testing.T) {
	v := New(1)
	v.SetBackgroundBlur(BackgroundBlur{Enabled: true, Type: " material ", Radius: -2})
	if got := v.BackgroundBlur(); got != (BackgroundBlur{Enabled: true, Type: "material", Radius: 0}) {
		t.Fatalf("BackgroundBlur() = %#v", got)
	}
	v.SetBackgroundColor("  #ffffff  ")
	if got := v.BackgroundColor(); got != "#ffffff" {
		t.Fatalf("BackgroundColor() = %q", got)
	}
	if err := v.SetBorderRadius(8); err != nil {
		t.Fatalf("SetBorderRadius() error = %v", err)
	}
	if v.BorderRadius() != 8 {
		t.Fatalf("BorderRadius() = %d, want 8", v.BorderRadius())
	}
	if err := v.SetBorderRadius(-1); !errors.Is(err, ErrInvalidBorderRadius) {
		t.Fatalf("SetBorderRadius(-1) error = %v, want ErrInvalidBorderRadius", err)
	}
	v.SetVisible(false)
	if v.Visible() {
		t.Fatal("Visible() = true, want false")
	}
}

func TestChildViewOrderingReparentingAndCopies(t *testing.T) {
	parent := New(1)
	a := New(2)
	b := New(3)
	c := New(4)
	if err := parent.AddChild(a, nil); err != nil {
		t.Fatalf("AddChild(a) error = %v", err)
	}
	if err := parent.AddChild(b, nil); err != nil {
		t.Fatalf("AddChild(b) error = %v", err)
	}
	index := 1
	if err := parent.AddChild(c, &index); err != nil {
		t.Fatalf("AddChild(c) error = %v", err)
	}
	if got := ids(parent.Children()); !reflect.DeepEqual(got, []int64{2, 4, 3}) {
		t.Fatalf("children = %#v", got)
	}
	if err := parent.AddChild(a, nil); err != nil {
		t.Fatalf("reorder AddChild(a) error = %v", err)
	}
	if got := ids(parent.Children()); !reflect.DeepEqual(got, []int64{4, 3, 2}) {
		t.Fatalf("children after reorder = %#v", got)
	}
	children := parent.Children()
	children[0] = nil
	if got := ids(parent.Children()); !reflect.DeepEqual(got, []int64{4, 3, 2}) {
		t.Fatalf("Children() returned mutable backing slice: %#v", got)
	}
	parent.RemoveChild(b)
	if got := ids(parent.Children()); !reflect.DeepEqual(got, []int64{4, 2}) {
		t.Fatalf("children after remove = %#v", got)
	}
	newParent := New(5)
	if err := newParent.AddChild(a, nil); err != nil {
		t.Fatalf("reparent AddChild(a) error = %v", err)
	}
	if got := ids(parent.Children()); !reflect.DeepEqual(got, []int64{4}) {
		t.Fatalf("old parent children after reparent = %#v", got)
	}
	if got := ids(newParent.Children()); !reflect.DeepEqual(got, []int64{2}) {
		t.Fatalf("new parent children after reparent = %#v", got)
	}
}

func TestAddChildValidation(t *testing.T) {
	v := New(1)
	if err := v.AddChild(nil, nil); !errors.Is(err, ErrInvalidChild) {
		t.Fatalf("AddChild(nil) error = %v, want ErrInvalidChild", err)
	}
	if err := v.AddChild(v, nil); !errors.Is(err, ErrInvalidChild) {
		t.Fatalf("AddChild(self) error = %v, want ErrInvalidChild", err)
	}
}

func ids(views []*View) []int64 {
	result := make([]int64, 0, len(views))
	for _, view := range views {
		if view == nil {
			result = append(result, -1)
			continue
		}
		result = append(result, view.ID())
	}
	return result
}
