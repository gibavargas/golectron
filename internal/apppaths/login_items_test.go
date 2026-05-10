package apppaths

import (
	"errors"
	"testing"
)

func TestLoginItemSettingsLifecycleAndArgsMatching(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{
		GOOS:       "windows",
		Home:       `C:\Users\me`,
		Temp:       `C:\Temp`,
		Executable: `C:\App\electron-go.exe`,
	})
	settings := LoginItemSettings{
		OpenAtLogin: true,
		Path:        `C:\App\Update.exe`,
		Args:        []string{"--processStart", "fixture.exe"},
	}
	if err := store.SetLoginItemSettings(settings); err != nil {
		t.Fatalf("SetLoginItemSettings() error = %v", err)
	}
	status, err := store.GetLoginItemSettings(LoginItemOptions{Path: settings.Path, Args: settings.Args})
	if err != nil {
		t.Fatalf("GetLoginItemSettings() error = %v", err)
	}
	if !status.OpenAtLogin || !status.ExecutableWillLaunchAtLogin {
		t.Fatalf("status = %#v, want openAtLogin and executableWillLaunchAtLogin", status)
	}
	if len(status.LaunchItems) != 1 || status.LaunchItems[0].Path != settings.Path || !status.LaunchItems[0].Enabled {
		t.Fatalf("LaunchItems = %#v, want enabled launch item", status.LaunchItems)
	}

	status, err = store.GetLoginItemSettings(LoginItemOptions{Path: settings.Path, Args: []string{"--different"}})
	if err != nil {
		t.Fatalf("GetLoginItemSettings(different args) error = %v", err)
	}
	if status.OpenAtLogin {
		t.Fatalf("OpenAtLogin = true for different args: %#v", status)
	}
	if !status.ExecutableWillLaunchAtLogin {
		t.Fatalf("ExecutableWillLaunchAtLogin = false for same executable with different args")
	}

	settings.OpenAtLogin = false
	if err := store.SetLoginItemSettings(settings); err != nil {
		t.Fatalf("SetLoginItemSettings(remove) error = %v", err)
	}
	status, _ = store.GetLoginItemSettings(LoginItemOptions{Path: settings.Path, Args: settings.Args})
	if status.OpenAtLogin || status.ExecutableWillLaunchAtLogin {
		t.Fatalf("status after removal = %#v, want disabled", status)
	}
}

func TestDarwinLoginItemStatusAndServiceValidation(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{
		GOOS:              "darwin",
		Home:              "/Users/me",
		Temp:              "/tmp",
		Executable:        "/Applications/Fixture.app/Contents/MacOS/Fixture",
		WasOpenedAtLogin:  true,
		WasOpenedAsHidden: true,
		RestoreState:      true,
	})
	if err := store.SetLoginItemSettings(LoginItemSettings{
		OpenAtLogin:  true,
		OpenAsHidden: true,
		Type:         LoginItemAgentService,
		ServiceName:  "com.example.fixture.agent",
	}); err != nil {
		t.Fatalf("SetLoginItemSettings(agent) error = %v", err)
	}
	status, err := store.GetLoginItemSettings(LoginItemOptions{
		Type:        LoginItemAgentService,
		ServiceName: "com.example.fixture.agent",
	})
	if err != nil {
		t.Fatalf("GetLoginItemSettings(agent) error = %v", err)
	}
	if !status.OpenAtLogin || !status.OpenAsHidden || !status.WasOpenedAtLogin || !status.WasOpenedAsHidden || !status.RestoreState {
		t.Fatalf("status = %#v, want macOS login item flags", status)
	}
	if status.Status != LoginItemStatusEnabled {
		t.Fatalf("Status = %q, want enabled", status.Status)
	}

	if err := store.SetLoginItemSettings(LoginItemSettings{OpenAtLogin: true, Type: LoginItemAgentService}); !errors.Is(err, ErrServiceNameRequired) {
		t.Fatalf("SetLoginItemSettings(missing serviceName) error = %v, want ErrServiceNameRequired", err)
	}
}

func TestLoginItemValidation(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: "/home/me", Temp: "/tmp", Executable: "/opt/electron-go"})
	if err := store.SetLoginItemSettings(LoginItemSettings{OpenAtLogin: true, Type: "bogus"}); !errors.Is(err, ErrInvalidLoginItemType) {
		t.Fatalf("SetLoginItemSettings(invalid type) error = %v, want ErrInvalidLoginItemType", err)
	}
	if err := store.SetLoginItemSettings(LoginItemSettings{OpenAtLogin: true, Path: "relative"}); !errors.Is(err, ErrPathNotAbsolute) {
		t.Fatalf("SetLoginItemSettings(relative path) error = %v, want ErrPathNotAbsolute", err)
	}
}
