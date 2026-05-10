package notification

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewNormalizesOptions(t *testing.T) {
	n, err := New(Options{
		Title:      "  Build complete  ",
		ID:         " build-1 ",
		GroupID:    " deploys ",
		GroupTitle: " Deploys ",
		Actions: []Action{
			{Text: "Open"},
			{Type: ActionDropdown, Text: "Snooze", Choices: []string{"5m"}},
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	options := n.Options()
	if options.Title != "Build complete" || options.ID != "build-1" || options.GroupID != "deploys" || options.GroupTitle != "Deploys" {
		t.Fatalf("normalized options = %#v", options)
	}
	if options.Urgency != UrgencyNormal {
		t.Fatalf("Urgency = %q, want normal", options.Urgency)
	}
	if options.Actions[0].Type != ActionButton || options.Actions[1].Choices[0] != "5m" {
		t.Fatalf("actions = %#v", options.Actions)
	}
	options.Actions[1].Choices[0] = "mutated"
	if got := n.Options().Actions[1].Choices[0]; got != "5m" {
		t.Fatalf("Options() did not copy action choices: %q", got)
	}
}

func TestNewRejectsInvalidOptions(t *testing.T) {
	cases := []Options{
		{},
		{Title: "Title", Urgency: "later"},
		{Title: "Title", Actions: []Action{{Type: "custom", Text: "Open"}}},
		{Title: "Title", Actions: []Action{{}}},
		{Title: "Title", Actions: []Action{{Type: ActionDropdown, Text: "Snooze"}}},
	}
	for _, tc := range cases {
		if _, err := New(tc); !errors.Is(err, ErrInvalidNotification) && !errors.Is(err, ErrInvalidAction) {
			t.Fatalf("New(%#v) error = %v, want validation error", tc, err)
		}
	}
}

func TestNotificationEvents(t *testing.T) {
	n, err := New(Options{
		Title:    "Message",
		HasReply: true,
		Actions:  []Action{{Text: "Open"}},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	n.Show()
	n.Click()
	if err := n.ActivateAction(0); err != nil {
		t.Fatalf("ActivateAction() error = %v", err)
	}
	if err := n.Reply("ok"); err != nil {
		t.Fatalf("Reply() error = %v", err)
	}
	n.Close(CloseReasonUser)
	want := []EventRecord{
		{Event: EventShow},
		{Event: EventClick},
		{Event: EventAction, ActionIndex: 0},
		{Event: EventReply, Reply: "ok"},
		{Event: EventClose, Reason: CloseReasonUser},
	}
	if got := n.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if err := n.ActivateAction(0); !errors.Is(err, ErrNotificationClosed) {
		t.Fatalf("ActivateAction(closed) error = %v, want ErrNotificationClosed", err)
	}
}

func TestNotificationActionAndReplyValidation(t *testing.T) {
	n, err := New(Options{Title: "Message", Actions: []Action{{Text: "Open"}}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	n.Show()
	if err := n.ActivateAction(1); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("ActivateAction(invalid) error = %v, want ErrInvalidAction", err)
	}
	if err := n.Reply("text"); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("Reply(disabled) error = %v, want ErrInvalidAction", err)
	}
}

func TestUnsignedAppEmitsFailed(t *testing.T) {
	n, err := New(Options{Title: "Message", UnsignedApp: true})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	n.Show()
	events := n.Events()
	if len(events) != 1 || events[0].Event != EventFailed {
		t.Fatalf("Events() = %#v, want failed", events)
	}
}

func TestHistoryRecordsShownNotificationsByID(t *testing.T) {
	var history History
	n, err := New(Options{Title: "Message", ID: "message-1", GroupID: "group"})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	history.Record(n)
	if len(history.GetHistory()) != 0 {
		t.Fatal("history recorded notification before show")
	}
	n.Show()
	history.Record(n)
	items := history.GetHistory()
	if len(items) != 1 || items[0].ID != "message-1" || items[0].GroupID != "group" {
		t.Fatalf("GetHistory() = %#v", items)
	}
	items[0].ID = "mutated"
	if got := history.GetHistory()[0].ID; got != "message-1" {
		t.Fatalf("history returned mutable item: %q", got)
	}
	if !history.RemoveByID("message-1") {
		t.Fatal("RemoveByID() = false, want true")
	}
	if len(history.GetHistory()) != 0 {
		t.Fatalf("history after remove = %#v", history.GetHistory())
	}
}
