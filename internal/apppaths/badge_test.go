package apppaths

import (
	"errors"
	"testing"
)

func TestBadgeCountDarwinSupportsCountsAndUnknownDot(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "darwin", Home: "/Users/me", Temp: "/tmp", Executable: "/Applications/Fixture.app/Contents/MacOS/Fixture"})
	if ok, err := store.SetBadgeCount(nil); err != nil || !ok {
		t.Fatalf("SetBadgeCount(nil) = %v, %v, want true nil", ok, err)
	}
	state := store.BadgeState()
	if !state.Supported || !state.Visible || !state.Unknown || state.Count != 0 {
		t.Fatalf("BadgeState(nil) = %#v, want visible unknown dot", state)
	}

	count := 12
	if ok, err := store.SetBadgeCount(&count); err != nil || !ok {
		t.Fatalf("SetBadgeCount(12) = %v, %v, want true nil", ok, err)
	}
	state = store.BadgeState()
	if !state.Visible || state.Unknown || state.Count != 12 || store.GetBadgeCount() != 12 {
		t.Fatalf("BadgeState(12) = %#v, want visible count", state)
	}
	count = 0
	store.SetBadgeCount(&count)
	state = store.BadgeState()
	if state.Visible || state.Unknown || state.Count != 0 {
		t.Fatalf("BadgeState(0) = %#v, want hidden zero", state)
	}
}

func TestBadgeCountLinuxRequiresUnity(t *testing.T) {
	linux := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: "/home/me", Temp: "/tmp", Executable: "/opt/electron-go"})
	count := 3
	if ok, err := linux.SetBadgeCount(&count); err != nil || ok {
		t.Fatalf("SetBadgeCount(non-Unity linux) = %v, %v, want false nil", ok, err)
	}
	if linux.BadgeState().Supported {
		t.Fatalf("BadgeState(non-Unity linux) = %#v, want unsupported", linux.BadgeState())
	}

	unity := NewStore("Fixture", "/app", Environment{GOOS: "linux", Home: "/home/me", Temp: "/tmp", Executable: "/opt/electron-go", UnityRunning: true})
	if !unity.IsUnityRunning() {
		t.Fatal("IsUnityRunning() = false, want true")
	}
	if ok, err := unity.SetBadgeCount(&count); err != nil || !ok {
		t.Fatalf("SetBadgeCount(Unity linux) = %v, %v, want true nil", ok, err)
	}
	if got := unity.GetBadgeCount(); got != 3 {
		t.Fatalf("GetBadgeCount() = %d, want 3", got)
	}
	if ok, err := unity.SetBadgeCount(nil); err != nil || !ok {
		t.Fatalf("SetBadgeCount(nil Unity linux) = %v, %v, want true nil", ok, err)
	}
	if unity.BadgeState().Visible {
		t.Fatalf("BadgeState(nil Unity linux) = %#v, want hidden", unity.BadgeState())
	}
}

func TestBadgeCountRejectsNegativeCounts(t *testing.T) {
	store := NewStore("Fixture", "/app", Environment{GOOS: "darwin", Home: "/Users/me", Temp: "/tmp", Executable: "/Applications/Fixture.app/Contents/MacOS/Fixture"})
	count := -1
	if _, err := store.SetBadgeCount(&count); !errors.Is(err, ErrInvalidBadgeCount) {
		t.Fatalf("SetBadgeCount(-1) error = %v, want ErrInvalidBadgeCount", err)
	}
}
