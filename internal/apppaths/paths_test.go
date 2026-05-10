package apppaths

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreDefaultsDarwinPaths(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	store := NewStore("Fixture", "/apps/fixture", Environment{
		GOOS:       "darwin",
		Home:       home,
		Temp:       "/tmp",
		Executable: "/apps/fixture/Electron",
	})

	assertPath(t, store, NameHome, home)
	assertPath(t, store, NameAppData, filepath.Join(home, "Library", "Application Support"))
	assertPath(t, store, NameUserData, filepath.Join(home, "Library", "Application Support", "Fixture"))
	assertPath(t, store, NameSession, filepath.Join(home, "Library", "Application Support", "Fixture"))
	assertPath(t, store, NameLogs, filepath.Join(home, "Library", "Logs", "Fixture"))
	if _, err := store.GetPath(NameAssets); !errors.Is(err, ErrUnknownPathName) {
		t.Fatalf("GetPath(assets) error = %v, want ErrUnknownPathName on darwin", err)
	}
}

func TestStoreDefaultsLinuxAndWindowsAppData(t *testing.T) {
	linux := NewStore("Fixture", "/app", Environment{
		GOOS:       "linux",
		Home:       "/home/me",
		Temp:       "/tmp",
		Executable: "/opt/electron/electron-go",
		Env:        map[string]string{"XDG_CONFIG_HOME": "/config"},
	})
	assertPath(t, linux, NameAppData, "/config")
	assertPath(t, linux, NameAssets, "/opt/electron")

	windows := NewStore("Fixture", `C:\App`, Environment{
		GOOS:       "windows",
		Home:       `C:\Users\me`,
		Temp:       `C:\Temp`,
		Executable: `C:\App\electron-go.exe`,
		Env:        map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming`},
	})
	assertPath(t, windows, NameAppData, `C:\Users\me\AppData\Roaming`)
	assertPath(t, windows, NameRecent, filepath.Join(`C:\Users\me\AppData\Roaming`, "Microsoft", "Windows", "Recent"))
}

func TestSetPathRequiresKnownExistingAbsoluteDirectory(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/bin/electron-go"})
	dir := t.TempDir()
	if err := store.SetPath(NameUserData, dir); err != nil {
		t.Fatalf("SetPath() error = %v", err)
	}
	assertPath(t, store, NameUserData, dir)
	assertPath(t, store, NameSession, dir)

	if err := store.SetPath("not-real", dir); !errors.Is(err, ErrUnknownPathName) {
		t.Fatalf("SetPath(unknown) error = %v, want ErrUnknownPathName", err)
	}
	if err := store.SetPath(NameUserData, "relative"); !errors.Is(err, ErrPathNotAbsolute) {
		t.Fatalf("SetPath(relative) error = %v, want ErrPathNotAbsolute", err)
	}
	if err := store.SetPath(NameUserData, filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrPathMissing) {
		t.Fatalf("SetPath(missing) error = %v, want ErrPathMissing", err)
	}
}

func TestSessionDataCanBeOverriddenSeparatelyFromUserData(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/bin/electron-go"})
	userData := t.TempDir()
	sessionData := t.TempDir()
	if err := store.SetPath(NameUserData, userData); err != nil {
		t.Fatalf("SetPath(userData) error = %v", err)
	}
	if err := store.SetPath(NameSession, sessionData); err != nil {
		t.Fatalf("SetPath(sessionData) error = %v", err)
	}
	assertPath(t, store, NameUserData, userData)
	assertPath(t, store, NameSession, sessionData)
}

func TestSetAppLogsPathCreatesDirectoryAndOverridesLogs(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/bin/electron-go"})
	logs := filepath.Join(t.TempDir(), "logs")
	if err := store.SetAppLogsPath(logs); err != nil {
		t.Fatalf("SetAppLogsPath() error = %v", err)
	}
	assertPath(t, store, NameLogs, logs)
}

func TestGetAppPath(t *testing.T) {
	store := NewStore("Fixture", "/apps/fixture", Environment{GOOS: "linux", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/bin/electron-go"})
	if got := store.GetAppPath(); got != "/apps/fixture" {
		t.Fatalf("GetAppPath() = %q, want /apps/fixture", got)
	}
}

func TestRecentDocumentsOnSupportedPlatforms(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "darwin", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/apps/fixture/Electron"})
	first := filepath.Join(t.TempDir(), "first.txt")
	second := filepath.Join(t.TempDir(), "second.txt")
	for _, doc := range []string{first, second, first} {
		if err := store.AddRecentDocument(doc); err != nil {
			t.Fatalf("AddRecentDocument(%s) error = %v", doc, err)
		}
	}
	got, err := store.GetRecentDocuments()
	if err != nil {
		t.Fatalf("GetRecentDocuments() error = %v", err)
	}
	want := []string{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetRecentDocuments() = %#v, want %#v", got, want)
	}
	got[0] = "mutated"
	got, _ = store.GetRecentDocuments()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetRecentDocuments() returned mutable backing slice: %#v", got)
	}
	if err := store.ClearRecentDocuments(); err != nil {
		t.Fatalf("ClearRecentDocuments() error = %v", err)
	}
	got, _ = store.GetRecentDocuments()
	if len(got) != 0 {
		t.Fatalf("GetRecentDocuments() after clear = %#v, want empty", got)
	}
}

func TestRecentDocumentsValidationAndUnsupportedPlatform(t *testing.T) {
	linux := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/bin/electron-go"})
	if err := linux.AddRecentDocument("/tmp/file.txt"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("AddRecentDocument(linux) error = %v, want ErrUnsupported", err)
	}
	if _, err := linux.GetRecentDocuments(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("GetRecentDocuments(linux) error = %v, want ErrUnsupported", err)
	}
	if err := linux.ClearRecentDocuments(); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ClearRecentDocuments(linux) error = %v, want ErrUnsupported", err)
	}

	darwin := NewStore("Fixture", "/app", Environment{GOOS: "darwin", Home: t.TempDir(), Temp: t.TempDir(), Executable: "/apps/fixture/Electron"})
	if err := darwin.AddRecentDocument(""); !errors.Is(err, ErrPathRequired) {
		t.Fatalf("AddRecentDocument(empty) error = %v, want ErrPathRequired", err)
	}
	if err := darwin.AddRecentDocument("relative.txt"); !errors.Is(err, ErrPathNotAbsolute) {
		t.Fatalf("AddRecentDocument(relative) error = %v, want ErrPathNotAbsolute", err)
	}
}

func assertPath(t *testing.T, store *Store, name, want string) {
	t.Helper()
	got, err := store.GetPath(name)
	if err != nil {
		t.Fatalf("GetPath(%s) error = %v", name, err)
	}
	if got != want {
		t.Fatalf("GetPath(%s) = %q, want %q", name, got, want)
	}
}
