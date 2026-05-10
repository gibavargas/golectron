package apppaths

import (
	"errors"
	"testing"
)

func TestProtocolClientRegistrationLifecycle(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{
		GOOS:       "windows",
		Home:       `C:\Users\me`,
		Temp:       `C:\Temp`,
		Executable: `C:\App\electron-go.exe`,
	})

	ok, err := store.SetAsDefaultProtocolClient("Electron-Test", "", []string{"--open-url"})
	if err != nil {
		t.Fatalf("SetAsDefaultProtocolClient() error = %v", err)
	}
	if !ok {
		t.Fatal("SetAsDefaultProtocolClient() = false, want true")
	}
	isDefault, err := store.IsDefaultProtocolClient("electron-test", "", []string{"--open-url"})
	if err != nil {
		t.Fatalf("IsDefaultProtocolClient() error = %v", err)
	}
	if !isDefault {
		t.Fatal("IsDefaultProtocolClient() = false, want true")
	}
	isDefault, err = store.IsDefaultProtocolClient("electron-test", "", []string{"--different"})
	if err != nil {
		t.Fatalf("IsDefaultProtocolClient(different args) error = %v", err)
	}
	if isDefault {
		t.Fatal("IsDefaultProtocolClient(different args) = true, want false")
	}
	removed, err := store.RemoveAsDefaultProtocolClient("electron-test", "", []string{"--open-url"})
	if err != nil {
		t.Fatalf("RemoveAsDefaultProtocolClient() error = %v", err)
	}
	if !removed {
		t.Fatal("RemoveAsDefaultProtocolClient() = false, want true")
	}
	isDefault, _ = store.IsDefaultProtocolClient("electron-test", "", []string{"--open-url"})
	if isDefault {
		t.Fatal("IsDefaultProtocolClient() = true after remove")
	}
}

func TestProtocolClientValidation(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: "/home/me", Temp: "/tmp", Executable: "/opt/electron-go"})
	tests := []struct {
		name     string
		protocol string
		path     string
		want     error
	}{
		{name: "empty", protocol: "", path: "", want: ErrProtocolRequired},
		{name: "url", protocol: "fixture://", path: "", want: ErrInvalidProtocol},
		{name: "leading digit", protocol: "1fixture", path: "", want: ErrInvalidProtocol},
		{name: "relative path", protocol: "fixture", path: "relative", want: ErrPathNotAbsolute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.SetAsDefaultProtocolClient(tt.protocol, tt.path, nil)
			if !errors.Is(err, tt.want) {
				t.Fatalf("SetAsDefaultProtocolClient() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestDarwinProtocolRegistrationRequiresDeclaredProtocol(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{
		GOOS:       "darwin",
		Home:       "/Users/me",
		Temp:       "/tmp",
		Executable: "/Applications/Fixture.app/Contents/MacOS/Fixture",
		DeclaredProtocols: map[string]struct{}{
			"fixture": {},
		},
	})
	ok, err := store.SetAsDefaultProtocolClient("fixture", "", nil)
	if err != nil {
		t.Fatalf("SetAsDefaultProtocolClient(declared) error = %v", err)
	}
	if !ok {
		t.Fatal("SetAsDefaultProtocolClient(declared) = false, want true")
	}
	if _, err := store.SetAsDefaultProtocolClient("missing", "", nil); !errors.Is(err, ErrProtocolUnavailable) {
		t.Fatalf("SetAsDefaultProtocolClient(missing) error = %v, want ErrProtocolUnavailable", err)
	}
}
