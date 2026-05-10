package apppaths

import (
	"errors"
	"fmt"
	"strings"
)

const (
	LoginItemMainAppService   = "mainAppService"
	LoginItemAgentService     = "agentService"
	LoginItemDaemonService    = "daemonService"
	LoginItemLoginItemService = "loginItemService"

	LoginItemStatusNotRegistered    = "not-registered"
	LoginItemStatusEnabled          = "enabled"
	LoginItemStatusRequiresApproval = "requires-approval"
	LoginItemStatusNotFound         = "not-found"
)

var (
	ErrInvalidLoginItemType = errors.New("invalid login item type")
	ErrServiceNameRequired  = errors.New("serviceName is required")
)

type LoginItemOptions struct {
	Type        string
	ServiceName string
	Path        string
	Args        []string
}

type LoginItemSettings struct {
	OpenAtLogin  bool
	OpenAsHidden bool
	Type         string
	ServiceName  string
	Path         string
	Args         []string
}

type LoginItemStatus struct {
	OpenAtLogin                 bool
	OpenAsHidden                bool
	WasOpenedAtLogin            bool
	WasOpenedAsHidden           bool
	RestoreState                bool
	Status                      string
	ExecutableWillLaunchAtLogin bool
	LaunchItems                 []LaunchItem
}

type LaunchItem struct {
	Name    string
	Path    string
	Args    []string
	Scope   string
	Enabled bool
}

type loginItemRecord struct {
	settings LoginItemSettings
}

func (s *Store) SetLoginItemSettings(settings LoginItemSettings) error {
	normalized, err := s.normalizeLoginItemSettings(settings)
	if err != nil {
		return err
	}
	key := normalized.loginItemKey()
	if !normalized.OpenAtLogin {
		delete(s.loginItems, key)
		return nil
	}
	s.loginItems[key] = loginItemRecord{settings: normalized}
	return nil
}

func (s *Store) GetLoginItemSettings(options LoginItemOptions) (LoginItemStatus, error) {
	normalized, err := s.normalizeLoginItemOptions(options)
	if err != nil {
		return LoginItemStatus{}, err
	}
	record, ok := s.loginItems[normalized.loginItemKey()]
	status := LoginItemStatus{
		Status: LoginItemStatusNotRegistered,
	}
	if !ok {
		status.ExecutableWillLaunchAtLogin = s.anyLoginItemForPath(normalized.Path)
		status.LaunchItems = s.launchItemsForPath(normalized.Path)
		return status, nil
	}
	status.OpenAtLogin = true
	status.OpenAsHidden = record.settings.OpenAsHidden
	status.WasOpenedAtLogin = s.env.WasOpenedAtLogin
	status.WasOpenedAsHidden = s.env.WasOpenedAsHidden
	status.RestoreState = s.env.RestoreState
	status.Status = LoginItemStatusEnabled
	status.ExecutableWillLaunchAtLogin = true
	status.LaunchItems = s.launchItemsForPath(normalized.Path)
	return status, nil
}

func (s *Store) normalizeLoginItemSettings(settings LoginItemSettings) (LoginItemSettings, error) {
	options := LoginItemOptions{
		Type:        settings.Type,
		ServiceName: settings.ServiceName,
		Path:        settings.Path,
		Args:        settings.Args,
	}
	normalized, err := s.normalizeLoginItemOptions(options)
	if err != nil {
		return LoginItemSettings{}, err
	}
	settings.Type = normalized.Type
	settings.ServiceName = normalized.ServiceName
	settings.Path = normalized.Path
	settings.Args = normalized.Args
	return settings, nil
}

func (s *Store) normalizeLoginItemOptions(options LoginItemOptions) (LoginItemOptions, error) {
	if options.Type == "" {
		options.Type = LoginItemMainAppService
	}
	if !validLoginItemType(options.Type) {
		return LoginItemOptions{}, fmt.Errorf("%w: %s", ErrInvalidLoginItemType, options.Type)
	}
	if options.Type != LoginItemMainAppService && strings.TrimSpace(options.ServiceName) == "" {
		return LoginItemOptions{}, ErrServiceNameRequired
	}
	if strings.TrimSpace(options.Path) == "" {
		options.Path = s.env.Executable
	}
	if !s.isAbsPath(options.Path) {
		return LoginItemOptions{}, fmt.Errorf("%w: %s", ErrPathNotAbsolute, options.Path)
	}
	options.Args = append([]string(nil), options.Args...)
	return options, nil
}

func (s *Store) anyLoginItemForPath(path string) bool {
	for _, record := range s.loginItems {
		if record.settings.Path == path {
			return true
		}
	}
	return false
}

func (s *Store) launchItemsForPath(path string) []LaunchItem {
	var items []LaunchItem
	for key, record := range s.loginItems {
		if record.settings.Path != path {
			continue
		}
		items = append(items, LaunchItem{
			Name:    key,
			Path:    record.settings.Path,
			Args:    append([]string(nil), record.settings.Args...),
			Scope:   "user",
			Enabled: true,
		})
	}
	return items
}

func (settings LoginItemSettings) loginItemKey() string {
	key := settings.Type + ":" + settings.ServiceName + ":" + settings.Path
	if len(settings.Args) > 0 {
		key += ":" + strings.Join(settings.Args, "\x00")
	}
	return key
}

func (options LoginItemOptions) loginItemKey() string {
	return LoginItemSettings{
		Type:        options.Type,
		ServiceName: options.ServiceName,
		Path:        options.Path,
		Args:        options.Args,
	}.loginItemKey()
}

func validLoginItemType(value string) bool {
	switch value {
	case LoginItemMainAppService, LoginItemAgentService, LoginItemDaemonService, LoginItemLoginItemService:
		return true
	default:
		return false
	}
}
