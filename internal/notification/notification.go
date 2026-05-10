package notification

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidNotification = errors.New("invalid notification")
	ErrInvalidAction       = errors.New("invalid notification action")
	ErrNotificationClosed  = errors.New("notification already closed")
)

type Urgency string

const (
	UrgencyLow      Urgency = "low"
	UrgencyNormal   Urgency = "normal"
	UrgencyCritical Urgency = "critical"
)

type CloseReason string

const (
	CloseReasonUnknown CloseReason = "unknown"
	CloseReasonUser    CloseReason = "user"
	CloseReasonTimeout CloseReason = "timeout"
	CloseReasonProgram CloseReason = "programmatic"
)

type ActionType string

const (
	ActionButton   ActionType = "button"
	ActionText     ActionType = "text"
	ActionDropdown ActionType = "dropdown"
)

type Action struct {
	Type        ActionType
	Text        string
	Placeholder string
	Choices     []string
}

type Options struct {
	Title            string
	Subtitle         string
	Body             string
	Silent           bool
	Icon             string
	Urgency          Urgency
	ID               string
	GroupID          string
	GroupTitle       string
	Actions          []Action
	ReplyPlaceholder string
	HasReply         bool
	UnsignedApp      bool
}

type Event string

const (
	EventShow   Event = "show"
	EventClick  Event = "click"
	EventClose  Event = "close"
	EventReply  Event = "reply"
	EventAction Event = "action"
	EventFailed Event = "failed"
)

type EventRecord struct {
	Event       Event
	ActionIndex int
	Reply       string
	Reason      CloseReason
	Error       string
}

type Notification struct {
	options Options
	shown   bool
	closed  bool
	events  []EventRecord
}

func New(options Options) (*Notification, error) {
	normalized, err := normalizeOptions(options)
	if err != nil {
		return nil, err
	}
	return &Notification{options: normalized}, nil
}

func (n *Notification) Options() Options {
	options := n.options
	options.Actions = cloneActions(options.Actions)
	return options
}

func (n *Notification) Show() {
	if n.options.UnsignedApp {
		n.events = append(n.events, EventRecord{Event: EventFailed, Error: "unsigned app cannot post notification"})
		return
	}
	if n.closed {
		n.events = append(n.events, EventRecord{Event: EventFailed, Error: ErrNotificationClosed.Error()})
		return
	}
	n.shown = true
	n.events = append(n.events, EventRecord{Event: EventShow})
}

func (n *Notification) Click() {
	if n.shown && !n.closed {
		n.events = append(n.events, EventRecord{Event: EventClick})
	}
}

func (n *Notification) ActivateAction(index int) error {
	if !n.shown || n.closed {
		return ErrNotificationClosed
	}
	if index < 0 || index >= len(n.options.Actions) {
		return fmt.Errorf("%w: action index", ErrInvalidAction)
	}
	n.events = append(n.events, EventRecord{Event: EventAction, ActionIndex: index})
	return nil
}

func (n *Notification) Reply(reply string) error {
	if !n.shown || n.closed {
		return ErrNotificationClosed
	}
	if !n.options.HasReply {
		return fmt.Errorf("%w: reply disabled", ErrInvalidAction)
	}
	n.events = append(n.events, EventRecord{Event: EventReply, Reply: reply})
	return nil
}

func (n *Notification) Close(reason CloseReason) {
	if n.closed {
		return
	}
	if reason == "" {
		reason = CloseReasonUnknown
	}
	n.closed = true
	n.events = append(n.events, EventRecord{Event: EventClose, Reason: reason})
}

func (n *Notification) Events() []EventRecord {
	return append([]EventRecord(nil), n.events...)
}

type History struct {
	items []Options
}

func (h *History) Record(n *Notification) {
	if n == nil || !n.shown || n.options.ID == "" {
		return
	}
	h.items = append(h.items, n.Options())
}

func (h *History) GetHistory() []Options {
	out := make([]Options, len(h.items))
	for i, item := range h.items {
		out[i] = item
		out[i].Actions = cloneActions(item.Actions)
	}
	return out
}

func (h *History) RemoveByID(id string) bool {
	id = strings.TrimSpace(id)
	for i, item := range h.items {
		if item.ID == id {
			h.items = append(h.items[:i], h.items[i+1:]...)
			return true
		}
	}
	return false
}

func normalizeOptions(options Options) (Options, error) {
	options.Title = strings.TrimSpace(options.Title)
	options.Subtitle = strings.TrimSpace(options.Subtitle)
	options.ID = strings.TrimSpace(options.ID)
	options.GroupID = strings.TrimSpace(options.GroupID)
	options.GroupTitle = strings.TrimSpace(options.GroupTitle)
	options.ReplyPlaceholder = strings.TrimSpace(options.ReplyPlaceholder)
	if options.Title == "" {
		return Options{}, fmt.Errorf("%w: title is required", ErrInvalidNotification)
	}
	if options.Urgency == "" {
		options.Urgency = UrgencyNormal
	}
	if options.Urgency != UrgencyLow && options.Urgency != UrgencyNormal && options.Urgency != UrgencyCritical {
		return Options{}, fmt.Errorf("%w: urgency", ErrInvalidNotification)
	}
	actions, err := normalizeActions(options.Actions)
	if err != nil {
		return Options{}, err
	}
	options.Actions = actions
	return options, nil
}

func normalizeActions(actions []Action) ([]Action, error) {
	normalized := make([]Action, len(actions))
	for i, action := range actions {
		action.Text = strings.TrimSpace(action.Text)
		action.Placeholder = strings.TrimSpace(action.Placeholder)
		if action.Type == "" {
			action.Type = ActionButton
		}
		if action.Type != ActionButton && action.Type != ActionText && action.Type != ActionDropdown {
			return nil, fmt.Errorf("%w: type", ErrInvalidAction)
		}
		if action.Text == "" {
			return nil, fmt.Errorf("%w: text is required", ErrInvalidAction)
		}
		if action.Type == ActionDropdown && len(action.Choices) == 0 {
			return nil, fmt.Errorf("%w: dropdown choices are required", ErrInvalidAction)
		}
		action.Choices = append([]string(nil), action.Choices...)
		normalized[i] = action
	}
	return normalized, nil
}

func cloneActions(actions []Action) []Action {
	out := make([]Action, len(actions))
	for i, action := range actions {
		out[i] = action
		out[i].Choices = append([]string(nil), action.Choices...)
	}
	return out
}
