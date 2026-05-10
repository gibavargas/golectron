package browserwindow

import (
	"errors"
	"math"
	"path/filepath"
	"testing"
)

func TestNormalizeOptionsAppliesElectronDefaults(t *testing.T) {
	got, err := NormalizeOptions(ConstructorOptions{})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	if got.Width != DefaultWidth || got.Height != DefaultHeight {
		t.Fatalf("bounds = %dx%d, want %dx%d", got.Width, got.Height, DefaultWidth, DefaultHeight)
	}
	if !got.Show {
		t.Fatal("Show = false, want true")
	}
	if !got.WebPreferences.DevTools {
		t.Fatal("DevTools = false, want true")
	}
	if got.WebPreferences.NodeIntegration {
		t.Fatal("NodeIntegration = true, want false")
	}
	if got.WebPreferences.NodeIntegrationInWorker {
		t.Fatal("NodeIntegrationInWorker = true, want false")
	}
	if !got.WebPreferences.ContextIsolation {
		t.Fatal("ContextIsolation = false, want true")
	}
	if !got.WebPreferences.BackgroundThrottling {
		t.Fatal("BackgroundThrottling = false, want true")
	}
	if !got.WebPreferences.FocusOnNavigation {
		t.Fatal("FocusOnNavigation = false, want true")
	}
	if got.WebPreferences.Offscreen.Enabled || got.WebPreferences.Offscreen.DeviceScaleFactor != 0 {
		t.Fatalf("Offscreen = %#v, want disabled zero value", got.WebPreferences.Offscreen)
	}
}

func TestNormalizeOptionsPreservesExplicitConstructorOptions(t *testing.T) {
	width := 1024
	height := 768
	x := 12
	y := 34
	show := false
	devTools := false
	contextIsolation := false
	backgroundThrottling := false
	focusOnNavigation := false
	deviceScaleFactor := 2.5
	preload := filepath.Join(t.TempDir(), "preload.js")
	got, err := NormalizeOptions(ConstructorOptions{
		Width:          &width,
		Height:         &height,
		X:              &x,
		Y:              &y,
		Show:           &show,
		UseContentSize: true,
		Center:         true,
		ParentID:       42,
		Modal:          true,
		Title:          "  Fixture  ",
		WebPreferences: WebPreferences{
			DevTools:                   &devTools,
			NodeIntegration:            true,
			NodeIntegrationInWorker:    true,
			NodeIntegrationInSubFrames: true,
			ContextIsolation:           &contextIsolation,
			Sandbox:                    true,
			Preload:                    preload,
			BackgroundThrottling:       &backgroundThrottling,
			FocusOnNavigation:          &focusOnNavigation,
			Offscreen: OffscreenPreferences{
				Enabled:           true,
				DeviceScaleFactor: &deviceScaleFactor,
			},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	if got.Width != width || got.Height != height || got.X == nil || *got.X != x || got.Y == nil || *got.Y != y {
		t.Fatalf("normalized bounds = %#v", got)
	}
	if got.Show || !got.UseContentSize || !got.Center || got.ParentID != 42 || !got.Modal {
		t.Fatalf("normalized window flags = %#v", got)
	}
	if got.Title != "Fixture" {
		t.Fatalf("Title = %q, want Fixture", got.Title)
	}
	prefs := got.WebPreferences
	if prefs.DevTools || !prefs.NodeIntegration || !prefs.NodeIntegrationInWorker || !prefs.NodeIntegrationInSubFrames || prefs.ContextIsolation || !prefs.Sandbox || prefs.Preload != preload || prefs.BackgroundThrottling || prefs.FocusOnNavigation {
		t.Fatalf("WebPreferences = %#v", prefs)
	}
	if !prefs.Offscreen.Enabled || prefs.Offscreen.DeviceScaleFactor != 2.5 {
		t.Fatalf("Offscreen = %#v, want enabled with deviceScaleFactor 2.5", prefs.Offscreen)
	}
}

func TestNormalizeOptionsRejectsInvalidBoundsAndRelationships(t *testing.T) {
	width := 0
	height := -1
	x := 10
	tests := []struct {
		name string
		opts ConstructorOptions
		want error
	}{
		{name: "width", opts: ConstructorOptions{Width: &width}, want: ErrInvalidBounds},
		{name: "height", opts: ConstructorOptions{Height: &height}, want: ErrInvalidBounds},
		{name: "x without y", opts: ConstructorOptions{X: &x}, want: ErrInvalidBounds},
		{name: "modal without parent", opts: ConstructorOptions{Modal: true}, want: ErrInvalidParent},
		{name: "relative preload", opts: ConstructorOptions{WebPreferences: WebPreferences{Preload: "preload.js"}}, want: ErrInvalidPreloadPath},
		{name: "offscreen disabled with scale", opts: ConstructorOptions{WebPreferences: WebPreferences{Offscreen: OffscreenPreferences{DeviceScaleFactor: floatPtr(1)}}}, want: ErrInvalidOffscreen},
		{name: "offscreen zero scale", opts: ConstructorOptions{WebPreferences: WebPreferences{Offscreen: OffscreenPreferences{Enabled: true, DeviceScaleFactor: floatPtr(0)}}}, want: ErrInvalidOffscreen},
		{name: "offscreen nan scale", opts: ConstructorOptions{WebPreferences: WebPreferences{Offscreen: OffscreenPreferences{Enabled: true, DeviceScaleFactor: floatPtr(math.NaN())}}}, want: ErrInvalidOffscreen},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizeOptions(tt.opts)
			if !errors.Is(err, tt.want) {
				t.Fatalf("NormalizeOptions() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNormalizeOptionsDefaultsOffscreenDeviceScaleFactor(t *testing.T) {
	got, err := NormalizeOptions(ConstructorOptions{
		WebPreferences: WebPreferences{
			Offscreen: OffscreenPreferences{Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	if !got.WebPreferences.Offscreen.Enabled {
		t.Fatal("Offscreen.Enabled = false, want true")
	}
	if got.WebPreferences.Offscreen.DeviceScaleFactor != 1.0 {
		t.Fatalf("DeviceScaleFactor = %v, want 1.0", got.WebPreferences.Offscreen.DeviceScaleFactor)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestNormalizeOptionsCopiesCoordinatePointers(t *testing.T) {
	x := 1
	y := 2
	got, err := NormalizeOptions(ConstructorOptions{X: &x, Y: &y})
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	x = 10
	y = 20
	if got.X == nil || *got.X != 1 || got.Y == nil || *got.Y != 2 {
		t.Fatalf("coordinates changed after caller mutation: %#v", got)
	}
}
