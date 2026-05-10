package menu

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidMenuItem = errors.New("invalid menu item")
	ErrMenuItemMissing = errors.New("menu item missing")
	ErrMenuDisposed    = errors.New("menu disposed")
	ErrTrayDisposed    = errors.New("tray disposed")
)

type ItemType string

const (
	ItemNormal    ItemType = "normal"
	ItemSeparator ItemType = "separator"
	ItemCheckbox  ItemType = "checkbox"
	ItemRadio     ItemType = "radio"
	ItemSubmenu   ItemType = "submenu"
)

type Role string

const (
	RoleAbout          Role = "about"
	RoleQuit           Role = "quit"
	RoleCopy           Role = "copy"
	RolePaste          Role = "paste"
	RoleToggleDevTools Role = "toggledevtools"
)

type ItemTemplate struct {
	ID          string
	Label       string
	Type        ItemType
	Role        Role
	Accelerator string
	Enabled     *bool
	Visible     *bool
	Checked     bool
	Submenu     []ItemTemplate
}

type Item struct {
	ID          string
	Label       string
	Type        ItemType
	Role        Role
	Accelerator string
	Enabled     bool
	Visible     bool
	Checked     bool
	Submenu     *Menu
}

type Menu struct {
	items    []*Item
	disposed bool
	events   []string
}

func BuildFromTemplate(template []ItemTemplate) (*Menu, error) {
	items := make([]*Item, 0, len(template))
	seenIDs := make(map[string]bool)
	for _, entry := range template {
		item, err := normalizeItem(entry)
		if err != nil {
			return nil, err
		}
		if item.ID != "" {
			if seenIDs[item.ID] {
				return nil, fmt.Errorf("%w: duplicate id %s", ErrInvalidMenuItem, item.ID)
			}
			seenIDs[item.ID] = true
		}
		items = append(items, item)
	}
	return &Menu{items: items}, nil
}

func (m *Menu) Items() []Item {
	out := make([]Item, 0, len(m.items))
	for _, item := range m.items {
		copy := *item
		out = append(out, copy)
	}
	return out
}

func (m *Menu) ItemByID(id string) (*Item, bool) {
	id = strings.TrimSpace(id)
	for _, item := range m.items {
		if item.ID == id {
			return item, true
		}
		if item.Submenu != nil {
			if found, ok := item.Submenu.ItemByID(id); ok {
				return found, true
			}
		}
	}
	return nil, false
}

func (m *Menu) UpdateItem(id string, update ItemTemplate) error {
	if m.disposed {
		return ErrMenuDisposed
	}
	item, ok := m.ItemByID(id)
	if !ok {
		return fmt.Errorf("%w: %s", ErrMenuItemMissing, id)
	}
	if update.Label != "" {
		item.Label = update.Label
	}
	if update.Enabled != nil {
		item.Enabled = *update.Enabled
	}
	if update.Visible != nil {
		item.Visible = *update.Visible
	}
	if update.Accelerator != "" {
		item.Accelerator = strings.TrimSpace(update.Accelerator)
	}
	if item.Type == ItemCheckbox || item.Type == ItemRadio {
		item.Checked = update.Checked
	}
	m.events = append(m.events, "update:"+id)
	return nil
}

func (m *Menu) Popup(id string) error {
	if m.disposed {
		return ErrMenuDisposed
	}
	if id != "" {
		item, ok := m.ItemByID(id)
		if !ok {
			return fmt.Errorf("%w: %s", ErrMenuItemMissing, id)
		}
		if !item.Enabled || !item.Visible || item.Type == ItemSeparator {
			return fmt.Errorf("%w: item cannot be activated", ErrInvalidMenuItem)
		}
		m.events = append(m.events, "click:"+id)
		return nil
	}
	m.events = append(m.events, "popup")
	return nil
}

func (m *Menu) Dispose() {
	m.disposed = true
	m.events = append(m.events, "dispose")
}

func (m *Menu) Events() []string {
	return append([]string(nil), m.events...)
}

type Tray struct {
	image       string
	tooltip     string
	title       string
	contextMenu *Menu
	events      []string
	disposed    bool
}

func NewTray(image string) (*Tray, error) {
	image = strings.TrimSpace(image)
	if image == "" {
		return nil, fmt.Errorf("%w: tray image is required", ErrInvalidMenuItem)
	}
	return &Tray{image: image}, nil
}

func (t *Tray) SetToolTip(tooltip string) error {
	if t.disposed {
		return ErrTrayDisposed
	}
	t.tooltip = tooltip
	t.events = append(t.events, "tooltip")
	return nil
}

func (t *Tray) SetTitle(title string) error {
	if t.disposed {
		return ErrTrayDisposed
	}
	t.title = title
	t.events = append(t.events, "title")
	return nil
}

func (t *Tray) SetContextMenu(menu *Menu) error {
	if t.disposed {
		return ErrTrayDisposed
	}
	t.contextMenu = menu
	t.events = append(t.events, "context-menu")
	return nil
}

func (t *Tray) Click() error {
	if t.disposed {
		return ErrTrayDisposed
	}
	t.events = append(t.events, "click")
	if t.contextMenu != nil {
		return t.contextMenu.Popup("")
	}
	return nil
}

func (t *Tray) Dispose() {
	t.disposed = true
	t.events = append(t.events, "dispose")
}

func (t *Tray) Events() []string {
	return append([]string(nil), t.events...)
}

func normalizeItem(entry ItemTemplate) (*Item, error) {
	itemType := entry.Type
	if itemType == "" {
		itemType = ItemNormal
	}
	if itemType != ItemNormal && itemType != ItemSeparator && itemType != ItemCheckbox && itemType != ItemRadio && itemType != ItemSubmenu {
		return nil, fmt.Errorf("%w: type %s", ErrInvalidMenuItem, itemType)
	}
	id := strings.TrimSpace(entry.ID)
	label := entry.Label
	if itemType != ItemSeparator && label == "" && entry.Role == "" {
		return nil, fmt.Errorf("%w: label or role is required", ErrInvalidMenuItem)
	}
	enabled := true
	if entry.Enabled != nil {
		enabled = *entry.Enabled
	}
	visible := true
	if entry.Visible != nil {
		visible = *entry.Visible
	}
	item := &Item{
		ID:          id,
		Label:       label,
		Type:        itemType,
		Role:        entry.Role,
		Accelerator: strings.TrimSpace(entry.Accelerator),
		Enabled:     enabled,
		Visible:     visible,
		Checked:     entry.Checked,
	}
	if len(entry.Submenu) > 0 {
		submenu, err := BuildFromTemplate(entry.Submenu)
		if err != nil {
			return nil, err
		}
		item.Type = ItemSubmenu
		item.Submenu = submenu
	}
	return item, nil
}
