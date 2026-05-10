package view

import (
	"errors"
	"fmt"
	"strings"
	"time"

	egwebcontents "github.com/gibavargas/electron-go/internal/webcontents"
)

type Event string

const EventBoundsChanged Event = "bounds-changed"

var (
	ErrInvalidBounds       = errors.New("invalid view bounds")
	ErrInvalidBorderRadius = errors.New("invalid border radius")
	ErrInvalidChild        = errors.New("invalid child view")
)

type Rect struct {
	X      int
	Y      int
	Width  int
	Height int
}

type SetBoundsOptions struct {
	Animate  bool
	Duration time.Duration
}

type BackgroundBlur struct {
	Enabled bool
	Type    string
	Radius  int
}

type Handler func(Event)

type View struct {
	id              int64
	bounds          Rect
	lastAnimation   SetBoundsOptions
	visible         bool
	backgroundColor string
	backgroundBlur  BackgroundBlur
	borderRadius    int
	parent          *View
	children        []*View
	handlers        map[Event][]Handler
	events          []Event
}

type WebContentsView struct {
	*View
	webContents *egwebcontents.WebContents
}

func New(id int64) *View {
	return &View{
		id:       id,
		visible:  true,
		handlers: make(map[Event][]Handler),
	}
}

func NewWebContentsView(id int64, webContentsID int64) *WebContentsView {
	return &WebContentsView{
		View:        New(id),
		webContents: egwebcontents.New(webContentsID),
	}
}

func (v *WebContentsView) WebContents() *egwebcontents.WebContents {
	return v.webContents
}

func (v *View) ID() int64 {
	return v.id
}

func (v *View) On(event Event, handler Handler) {
	if handler == nil {
		return
	}
	v.handlers[event] = append(v.handlers[event], handler)
}

func (v *View) SetBounds(bounds Rect, opts SetBoundsOptions) error {
	if err := validateBounds(bounds); err != nil {
		return err
	}
	if opts.Duration < 0 {
		return fmt.Errorf("%w: animation duration must not be negative", ErrInvalidBounds)
	}
	if v.bounds == bounds && v.lastAnimation == opts {
		return nil
	}
	v.bounds = bounds
	v.lastAnimation = opts
	v.emit(EventBoundsChanged)
	return nil
}

func (v *View) Bounds() Rect {
	return v.bounds
}

func (v *View) LastAnimation() SetBoundsOptions {
	return v.lastAnimation
}

func (v *View) SetBackgroundBlur(blur BackgroundBlur) {
	blur.Type = strings.TrimSpace(blur.Type)
	if blur.Radius < 0 {
		blur.Radius = 0
	}
	v.backgroundBlur = blur
}

func (v *View) BackgroundBlur() BackgroundBlur {
	return v.backgroundBlur
}

func (v *View) SetBackgroundColor(color string) {
	v.backgroundColor = strings.TrimSpace(color)
}

func (v *View) BackgroundColor() string {
	return v.backgroundColor
}

func (v *View) SetBorderRadius(radius int) error {
	if radius < 0 {
		return fmt.Errorf("%w: %d", ErrInvalidBorderRadius, radius)
	}
	v.borderRadius = radius
	return nil
}

func (v *View) BorderRadius() int {
	return v.borderRadius
}

func (v *View) SetVisible(visible bool) {
	v.visible = visible
}

func (v *View) Visible() bool {
	return v.visible
}

func (v *View) AddChild(child *View, index *int) error {
	if child == nil || child == v {
		return ErrInvalidChild
	}
	if child.parent != nil {
		child.parent.RemoveChild(child)
	}
	removeChildRef(&v.children, child)
	insertAt := len(v.children)
	if index != nil {
		insertAt = *index
		if insertAt < 0 {
			insertAt = 0
		}
		if insertAt > len(v.children) {
			insertAt = len(v.children)
		}
	}
	v.children = append(v.children, nil)
	copy(v.children[insertAt+1:], v.children[insertAt:])
	v.children[insertAt] = child
	child.parent = v
	return nil
}

func (v *View) RemoveChild(child *View) {
	if child == nil {
		return
	}
	if removeChildRef(&v.children, child) {
		child.parent = nil
	}
}

func (v *View) Children() []*View {
	return append([]*View(nil), v.children...)
}

func (v *View) Events() []Event {
	return append([]Event(nil), v.events...)
}

func (v *View) emit(event Event) {
	v.events = append(v.events, event)
	for _, handler := range v.handlers[event] {
		handler(event)
	}
}

func validateBounds(bounds Rect) error {
	if bounds.Width < 0 {
		return fmt.Errorf("%w: width must not be negative", ErrInvalidBounds)
	}
	if bounds.Height < 0 {
		return fmt.Errorf("%w: height must not be negative", ErrInvalidBounds)
	}
	return nil
}

func removeChildRef(children *[]*View, child *View) bool {
	for i, existing := range *children {
		if existing == child {
			copy((*children)[i:], (*children)[i+1:])
			*children = (*children)[:len(*children)-1]
			return true
		}
	}
	return false
}
