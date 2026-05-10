package menu

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildFromTemplateNormalizesItems(t *testing.T) {
	disabled := false
	menu, err := BuildFromTemplate([]ItemTemplate{
		{ID: "open", Label: " Open ", Accelerator: "CmdOrCtrl+O"},
		{Type: ItemSeparator},
		{ID: "copy", Role: RoleCopy},
		{ID: "toggle", Label: "Enabled", Type: ItemCheckbox, Checked: true, Enabled: &disabled},
		{ID: "view", Label: "View", Submenu: []ItemTemplate{{ID: "devtools", Role: RoleToggleDevTools}}},
	})
	if err != nil {
		t.Fatalf("BuildFromTemplate() error = %v", err)
	}
	items := menu.Items()
	if len(items) != 5 {
		t.Fatalf("Items() len = %d, want 5", len(items))
	}
	if items[0].Label != " Open " || items[0].Type != ItemNormal || !items[0].Enabled || !items[0].Visible {
		t.Fatalf("first item = %#v", items[0])
	}
	if items[3].Enabled || !items[3].Checked {
		t.Fatalf("checkbox item = %#v", items[3])
	}
	if item, ok := menu.ItemByID("devtools"); !ok || item.Role != RoleToggleDevTools {
		t.Fatalf("submenu lookup = %#v ok=%v", item, ok)
	}
}

func TestBuildFromTemplateRejectsInvalidItems(t *testing.T) {
	cases := []ItemTemplate{
		{Type: "custom", Label: "Bad"},
		{ID: "missing"},
	}
	for _, tc := range cases {
		if _, err := BuildFromTemplate([]ItemTemplate{tc}); !errors.Is(err, ErrInvalidMenuItem) {
			t.Fatalf("BuildFromTemplate(%#v) error = %v, want ErrInvalidMenuItem", tc, err)
		}
	}
	if _, err := BuildFromTemplate([]ItemTemplate{{ID: "x", Label: "One"}, {ID: "x", Label: "Two"}}); !errors.Is(err, ErrInvalidMenuItem) {
		t.Fatalf("BuildFromTemplate(duplicate) error = %v, want ErrInvalidMenuItem", err)
	}
}

func TestMenuUpdatePopupAndDispose(t *testing.T) {
	enabled := false
	visible := false
	menu, err := BuildFromTemplate([]ItemTemplate{{ID: "open", Label: "Open"}})
	if err != nil {
		t.Fatalf("BuildFromTemplate() error = %v", err)
	}
	if err := menu.UpdateItem("open", ItemTemplate{Label: "Open File", Enabled: &enabled, Visible: &visible}); err != nil {
		t.Fatalf("UpdateItem() error = %v", err)
	}
	item, _ := menu.ItemByID("open")
	if item.Label != "Open File" || item.Enabled || item.Visible {
		t.Fatalf("updated item = %#v", item)
	}
	if err := menu.Popup("open"); !errors.Is(err, ErrInvalidMenuItem) {
		t.Fatalf("Popup(disabled) error = %v, want ErrInvalidMenuItem", err)
	}
	enabled = true
	visible = true
	if err := menu.UpdateItem("open", ItemTemplate{Enabled: &enabled, Visible: &visible}); err != nil {
		t.Fatalf("UpdateItem(enable) error = %v", err)
	}
	if err := menu.Popup("open"); err != nil {
		t.Fatalf("Popup() error = %v", err)
	}
	wantEvents := []string{"update:open", "update:open", "click:open"}
	if got := menu.Events(); !reflect.DeepEqual(got, wantEvents) {
		t.Fatalf("Events() = %#v, want %#v", got, wantEvents)
	}
	menu.Dispose()
	if err := menu.UpdateItem("open", ItemTemplate{Label: "After"}); !errors.Is(err, ErrMenuDisposed) {
		t.Fatalf("UpdateItem(disposed) error = %v, want ErrMenuDisposed", err)
	}
}

func TestTrayLifecycleAndContextMenu(t *testing.T) {
	contextMenu, err := BuildFromTemplate([]ItemTemplate{{ID: "open", Label: "Open"}})
	if err != nil {
		t.Fatalf("BuildFromTemplate() error = %v", err)
	}
	tray, err := NewTray("icon.png")
	if err != nil {
		t.Fatalf("NewTray() error = %v", err)
	}
	if err := tray.SetToolTip("Tooltip"); err != nil {
		t.Fatalf("SetToolTip() error = %v", err)
	}
	if err := tray.SetTitle("Title"); err != nil {
		t.Fatalf("SetTitle() error = %v", err)
	}
	if err := tray.SetContextMenu(contextMenu); err != nil {
		t.Fatalf("SetContextMenu() error = %v", err)
	}
	if err := tray.Click(); err != nil {
		t.Fatalf("Click() error = %v", err)
	}
	if got := contextMenu.Events(); !reflect.DeepEqual(got, []string{"popup"}) {
		t.Fatalf("context menu events = %#v", got)
	}
	wantTrayEvents := []string{"tooltip", "title", "context-menu", "click"}
	if got := tray.Events(); !reflect.DeepEqual(got, wantTrayEvents) {
		t.Fatalf("tray events = %#v, want %#v", got, wantTrayEvents)
	}
	tray.Dispose()
	if err := tray.Click(); !errors.Is(err, ErrTrayDisposed) {
		t.Fatalf("Click(disposed) error = %v, want ErrTrayDisposed", err)
	}
}

func TestNewTrayRejectsMissingImage(t *testing.T) {
	if _, err := NewTray(" "); !errors.Is(err, ErrInvalidMenuItem) {
		t.Fatalf("NewTray(empty) error = %v, want ErrInvalidMenuItem", err)
	}
}
