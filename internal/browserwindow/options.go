package browserwindow

import (
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
)

const (
	DefaultWidth  = 800
	DefaultHeight = 600
)

var (
	ErrInvalidBounds      = errors.New("invalid browser window bounds")
	ErrInvalidPreloadPath = errors.New("invalid preload path")
	ErrInvalidParent      = errors.New("invalid parent window")
	ErrInvalidOffscreen   = errors.New("invalid offscreen preferences")
)

type ConstructorOptions struct {
	Width          *int
	Height         *int
	X              *int
	Y              *int
	UseContentSize bool
	Center         bool
	Show           *bool
	ParentID       int64
	Modal          bool
	Title          string
	WebPreferences WebPreferences
}

type WebPreferences struct {
	DevTools                   *bool
	NodeIntegration            bool
	NodeIntegrationInWorker    bool
	NodeIntegrationInSubFrames bool
	ContextIsolation           *bool
	Sandbox                    bool
	Preload                    string
	BackgroundThrottling       *bool
	FocusOnNavigation          *bool
	Offscreen                  OffscreenPreferences
}

type OffscreenPreferences struct {
	Enabled           bool
	DeviceScaleFactor *float64
}

type NormalizedOptions struct {
	Width          int
	Height         int
	X              *int
	Y              *int
	UseContentSize bool
	Center         bool
	Show           bool
	ParentID       int64
	Modal          bool
	Title          string
	WebPreferences NormalizedWebPreferences
}

type NormalizedWebPreferences struct {
	DevTools                   bool
	NodeIntegration            bool
	NodeIntegrationInWorker    bool
	NodeIntegrationInSubFrames bool
	ContextIsolation           bool
	Sandbox                    bool
	Preload                    string
	BackgroundThrottling       bool
	FocusOnNavigation          bool
	Offscreen                  NormalizedOffscreenPreferences
}

type NormalizedOffscreenPreferences struct {
	Enabled           bool
	DeviceScaleFactor float64
}

func NormalizeOptions(opts ConstructorOptions) (NormalizedOptions, error) {
	width := DefaultWidth
	if opts.Width != nil {
		width = *opts.Width
	}
	height := DefaultHeight
	if opts.Height != nil {
		height = *opts.Height
	}
	if width <= 0 {
		return NormalizedOptions{}, fmt.Errorf("%w: width must be positive", ErrInvalidBounds)
	}
	if height <= 0 {
		return NormalizedOptions{}, fmt.Errorf("%w: height must be positive", ErrInvalidBounds)
	}
	if (opts.X == nil) != (opts.Y == nil) {
		return NormalizedOptions{}, fmt.Errorf("%w: x and y must be provided together", ErrInvalidBounds)
	}
	if opts.Modal && opts.ParentID <= 0 {
		return NormalizedOptions{}, fmt.Errorf("%w: modal windows require a parent", ErrInvalidParent)
	}
	webPreferences, err := normalizeWebPreferences(opts.WebPreferences)
	if err != nil {
		return NormalizedOptions{}, err
	}
	show := true
	if opts.Show != nil {
		show = *opts.Show
	}
	return NormalizedOptions{
		Width:          width,
		Height:         height,
		X:              cloneInt(opts.X),
		Y:              cloneInt(opts.Y),
		UseContentSize: opts.UseContentSize,
		Center:         opts.Center,
		Show:           show,
		ParentID:       opts.ParentID,
		Modal:          opts.Modal,
		Title:          strings.TrimSpace(opts.Title),
		WebPreferences: webPreferences,
	}, nil
}

func normalizeWebPreferences(prefs WebPreferences) (NormalizedWebPreferences, error) {
	if prefs.Preload != "" && !filepath.IsAbs(prefs.Preload) {
		return NormalizedWebPreferences{}, fmt.Errorf("%w: preload must be absolute", ErrInvalidPreloadPath)
	}
	devTools := true
	if prefs.DevTools != nil {
		devTools = *prefs.DevTools
	}
	contextIsolation := true
	if prefs.ContextIsolation != nil {
		contextIsolation = *prefs.ContextIsolation
	}
	backgroundThrottling := true
	if prefs.BackgroundThrottling != nil {
		backgroundThrottling = *prefs.BackgroundThrottling
	}
	focusOnNavigation := true
	if prefs.FocusOnNavigation != nil {
		focusOnNavigation = *prefs.FocusOnNavigation
	}
	offscreen, err := normalizeOffscreenPreferences(prefs.Offscreen)
	if err != nil {
		return NormalizedWebPreferences{}, err
	}
	return NormalizedWebPreferences{
		DevTools:                   devTools,
		NodeIntegration:            prefs.NodeIntegration,
		NodeIntegrationInWorker:    prefs.NodeIntegrationInWorker,
		NodeIntegrationInSubFrames: prefs.NodeIntegrationInSubFrames,
		ContextIsolation:           contextIsolation,
		Sandbox:                    prefs.Sandbox,
		Preload:                    prefs.Preload,
		BackgroundThrottling:       backgroundThrottling,
		FocusOnNavigation:          focusOnNavigation,
		Offscreen:                  offscreen,
	}, nil
}

func normalizeOffscreenPreferences(prefs OffscreenPreferences) (NormalizedOffscreenPreferences, error) {
	if !prefs.Enabled {
		if prefs.DeviceScaleFactor != nil {
			return NormalizedOffscreenPreferences{}, fmt.Errorf("%w: deviceScaleFactor requires offscreen rendering", ErrInvalidOffscreen)
		}
		return NormalizedOffscreenPreferences{}, nil
	}
	deviceScaleFactor := 1.0
	if prefs.DeviceScaleFactor != nil {
		deviceScaleFactor = *prefs.DeviceScaleFactor
	}
	if deviceScaleFactor <= 0 || math.IsInf(deviceScaleFactor, 0) || math.IsNaN(deviceScaleFactor) {
		return NormalizedOffscreenPreferences{}, fmt.Errorf("%w: deviceScaleFactor must be positive", ErrInvalidOffscreen)
	}
	return NormalizedOffscreenPreferences{
		Enabled:           true,
		DeviceScaleFactor: deviceScaleFactor,
	}, nil
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
