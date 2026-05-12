package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"reflect"
	goruntime "runtime"
	"testing"

	egruntime "github.com/gibavargas/electron-go/internal/runtime"
)

func TestRunCompatJSONPrintsLedger(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--compat-json"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--compat-json) exit = %d, want 0", code)
	}
	var payload struct {
		Completion string `json:"completion"`
		Items      []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--compat-json output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Completion != "complete" {
		t.Fatalf("completion = %q, want complete", payload.Completion)
	}
	if len(payload.Items) != 49 {
		t.Fatalf("items = %d, want 49", len(payload.Items))
	}
}

func TestRunCheckParityPassesWhenLedgerComplete(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--check-parity"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--check-parity) exit = %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("parity complete")) {
		t.Fatalf("stdout = %q, want parity complete", stdout.String())
	}
}

func TestRunE2EAuditPrintsMissingCompatibleCoverage(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--e2e-audit"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--e2e-audit) exit = %d, want 0", code)
	}
	var payload struct {
		Pass       bool     `json:"pass"`
		Missing    []string `json:"missing_e2e_evidence"`
		Incomplete []string `json:"incomplete_items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--e2e-audit output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Missing == nil {
		t.Fatalf("missing_e2e_evidence = nil, want JSON array")
	}
	if payload.Incomplete == nil {
		t.Fatalf("incomplete_items = nil, want JSON array")
	}
	if !payload.Pass {
		t.Fatal("pass = false while ledger is complete")
	}
	if len(payload.Incomplete) != 0 {
		t.Fatalf("incomplete_items = %#v, want empty", payload.Incomplete)
	}
}

func TestRunRuntimeParityAuditReportsGatedRuntimeConformance(t *testing.T) {
	t.Setenv("ELECTRON_GO_ENABLE_IPC_CONFORMANCE", "")
	t.Setenv("ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE", "")

	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--runtime-parity-audit"}, &stdout, nil)
	if code != 1 {
		t.Fatalf("run(--runtime-parity-audit) exit = %d, want 1 while gates are disabled", code)
	}
	var payload struct {
		Pass  bool `json:"pass"`
		Gates []struct {
			Name    string `json:"name"`
			Env     string `json:"env"`
			Enabled bool   `json:"enabled"`
		} `json:"gates"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--runtime-parity-audit output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Pass {
		t.Fatal("pass = true while runtime conformance gates are disabled")
	}
	if len(payload.Gates) != 2 {
		t.Fatalf("gates = %d, want 2", len(payload.Gates))
	}
	for _, gate := range payload.Gates {
		if gate.Name == "" || gate.Env == "" {
			t.Fatalf("gate has missing identity: %#v", gate)
		}
		if gate.Enabled {
			t.Fatalf("gate %s enabled = true, want false", gate.Name)
		}
	}
}

func TestRunRuntimeParityAuditPassesWhenRuntimeGatesEnabled(t *testing.T) {
	t.Setenv("ELECTRON_GO_ENABLE_IPC_CONFORMANCE", "1")
	t.Setenv("ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE", "1")

	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--runtime-parity-audit"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--runtime-parity-audit) exit = %d, want 0 when gates are enabled", code)
	}
	var payload struct {
		Pass bool `json:"pass"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--runtime-parity-audit output is not JSON: %v\n%s", err, stdout.String())
	}
	if !payload.Pass {
		t.Fatal("pass = false while runtime conformance gates are enabled")
	}
}

func TestRunGoalAuditReportsRemainingObjectiveGaps(t *testing.T) {
	t.Setenv("ELECTRON_GO_ENABLE_IPC_CONFORMANCE", "")
	t.Setenv("ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE", "")

	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--goal-audit"}, &stdout, nil)
	if code != 1 {
		t.Fatalf("run(--goal-audit) exit = %d, want 1 while objective remains incomplete", code)
	}
	var payload struct {
		Pass   bool `json:"pass"`
		Ledger struct {
			Pass       bool `json:"pass"`
			Compatible int  `json:"compatible"`
			Total      int  `json:"total"`
		} `json:"ledger"`
		Runtime struct {
			Pass bool `json:"pass"`
		} `json:"runtime"`
		Performance struct {
			Pass                         bool    `json:"pass"`
			TargetElectronGoOverElectron float64 `json:"target_electron_go_over_electron"`
			LatestElectronGoOverElectron float64 `json:"latest_electron_go_over_electron"`
		} `json:"performance"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--goal-audit output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Pass {
		t.Fatal("pass = true while runtime and performance targets are incomplete")
	}
	if !payload.Ledger.Pass || payload.Ledger.Compatible != 49 || payload.Ledger.Total != 49 {
		t.Fatalf("ledger audit = %#v, want 49/49 pass", payload.Ledger)
	}
	if payload.Runtime.Pass {
		t.Fatal("runtime pass = true while runtime gates are disabled")
	}
	if payload.Performance.Pass {
		t.Fatal("performance pass = true before 0.5 startup ratio target")
	}
	if payload.Performance.TargetElectronGoOverElectron != 0.5 || payload.Performance.LatestElectronGoOverElectron <= 0.5 {
		t.Fatalf("performance audit = %#v, want latest ratio above 0.5 target", payload.Performance)
	}
}

func TestGoalPerformanceEvidenceLoadsCheckedInBenchmarkEvidence(t *testing.T) {
	got := goalPerformanceEvidence()
	if got.TargetElectronGoOverElectron != 0.5 {
		t.Fatalf("target ratio = %v, want 0.5", got.TargetElectronGoOverElectron)
	}
	if got.LatestElectronGoOverElectron != 0.7593457943925234 {
		t.Fatalf("latest ratio = %v, want 0.7593457943925234", got.LatestElectronGoOverElectron)
	}
	if got.LatestRunURL == "" || got.LatestCommit == "" || got.Metric != "duration_median_ms" {
		t.Fatalf("benchmark evidence = %#v, want run URL, commit, and duration_median_ms metric", got)
	}
}

func TestRunClipboardCheckReportsTextRoundTrip(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--clipboard-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--clipboard-check) exit = %d, want 0", code)
	}

	var payload struct {
		TextRoundTrip bool   `json:"textRoundTrip"`
		Text          string `json:"text"`
		Error         string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--clipboard-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("clipboard error = %q", payload.Error)
	}
	if !payload.TextRoundTrip || payload.Text != "electron-go clipboard" {
		t.Fatalf("clipboard report = %#v, want text round trip", payload)
	}
}

func TestRunClipboardFormatsCheckReportsRichFormatRoundTrips(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--clipboard-formats-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--clipboard-formats-check) exit = %d, want 0", code)
	}

	var payload struct {
		HTMLContainsPayload bool   `json:"htmlContainsPayload"`
		RTFRoundTrip        bool   `json:"rtfRoundTrip"`
		RTFBytes            int    `json:"rtfBytes"`
		BufferRoundTrip     bool   `json:"bufferRoundTrip"`
		BufferBytes         int    `json:"bufferBytes"`
		Error               string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--clipboard-formats-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("clipboard formats error = %q", payload.Error)
	}
	if !payload.HTMLContainsPayload || !payload.RTFRoundTrip || !payload.BufferRoundTrip {
		t.Fatalf("clipboard format report = %#v, want HTML payload, RTF round trip, and buffer round trip", payload)
	}
	if payload.RTFBytes == 0 || payload.BufferBytes != 3 {
		t.Fatalf("clipboard format payload = %#v, want nonempty RTF and 3-byte buffer", payload)
	}
}

func TestRunShellCheckReportsMissingOpenPath(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--shell-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--shell-check) exit = %d, want 0", code)
	}

	var payload struct {
		MissingPathRejected  bool   `json:"missingPathRejected"`
		ErrorMessageNonempty bool   `json:"errorMessageNonempty"`
		Error                string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--shell-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("shell error = %q", payload.Error)
	}
	if !payload.MissingPathRejected || !payload.ErrorMessageNonempty {
		t.Fatalf("shell report = %#v, want missing openPath rejection with message", payload)
	}
}

func TestRunContentTracingCheckReportsLifecycle(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--content-tracing-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--content-tracing-check) exit = %d, want 0", code)
	}

	var payload struct {
		Started               bool     `json:"started"`
		Stopped               bool     `json:"stopped"`
		RequestedPathReturned bool     `json:"requestedPathReturned"`
		Categories            []string `json:"categories"`
		HeapProfiling         bool     `json:"heapProfiling"`
		Error                 string   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--content-tracing-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("contentTracing error = %q", payload.Error)
	}
	if !payload.Started || !payload.Stopped || !payload.RequestedPathReturned {
		t.Fatalf("contentTracing report = %#v, want completed start/stop with requested path", payload)
	}
	if len(payload.Categories) != 1 || payload.Categories[0] != "electron" {
		t.Fatalf("contentTracing categories = %#v, want [electron]", payload.Categories)
	}
	if payload.HeapProfiling {
		t.Fatalf("contentTracing heapProfiling = true, want false")
	}
}

func TestRunCrashReporterCheckReportsParameters(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--crash-reporter-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--crash-reporter-check) exit = %d, want 0", code)
	}

	var payload struct {
		Started              bool   `json:"started"`
		UploadInitial        bool   `json:"uploadInitial"`
		UploadAfterSet       bool   `json:"uploadAfterSet"`
		ExtraInitial         bool   `json:"extraInitial"`
		ExtraAdded           bool   `json:"extraAdded"`
		ExtraRemoved         bool   `json:"extraRemoved"`
		UploadedReportsEmpty bool   `json:"uploadedReportsEmpty"`
		LastReportNull       bool   `json:"lastReportNull"`
		Error                string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--crash-reporter-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("crashReporter error = %q", payload.Error)
	}
	if !payload.Started || payload.UploadInitial || !payload.UploadAfterSet || !payload.ExtraInitial || !payload.ExtraAdded || !payload.ExtraRemoved || !payload.UploadedReportsEmpty || !payload.LastReportNull {
		t.Fatalf("crashReporter report = %#v, want start, extra parameter, upload toggle, and empty report state", payload)
	}
}

func TestRunBrowserWindowOptionsCheckReportsConstructorState(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--browser-window-options-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--browser-window-options-check) exit = %d, want 0", code)
	}

	var payload struct {
		Width                    int    `json:"width"`
		Height                   int    `json:"height"`
		Visible                  bool   `json:"visible"`
		Title                    string `json:"title"`
		DevTools                 bool   `json:"devTools"`
		DefaultDevTools          bool   `json:"defaultDevTools"`
		DefaultContextIsolation  bool   `json:"defaultContextIsolation"`
		ExplicitContextIsolation bool   `json:"explicitContextIsolation"`
		ExplicitNodeIntegration  bool   `json:"explicitNodeIntegration"`
		ModalConstructed         bool   `json:"modalConstructed"`
		Error                    string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--browser-window-options-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("browser window options error = %q", payload.Error)
	}
	if payload.Width != 640 || payload.Height != 480 || payload.Visible || payload.Title != "Fixture" || payload.DevTools {
		t.Fatalf("browser window options report = %#v, want hidden 640x480 Fixture with devTools disabled", payload)
	}
	if !payload.DefaultDevTools || !payload.DefaultContextIsolation || payload.ExplicitContextIsolation || !payload.ExplicitNodeIntegration || !payload.ModalConstructed {
		t.Fatalf("browser window webPreferences/options report = %#v, want default prefs and explicit constructor state", payload)
	}
}

func TestRunBrowserWindowMethodsCheckReportsStateTransitions(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--browser-window-methods-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--browser-window-methods-check) exit = %d, want 0", code)
	}

	var payload struct {
		InitialVisible bool     `json:"initialVisible"`
		AfterShow      bool     `json:"afterShow"`
		AfterHide      bool     `json:"afterHide"`
		EventOrder     []string `json:"eventOrder"`
		Bounds         struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"bounds"`
		NormalBounds struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"normalBounds"`
		EnabledAfterDisable       bool   `json:"enabledAfterDisable"`
		EnabledAfterEnable        bool   `json:"enabledAfterEnable"`
		MinimizedAfterMinimize    bool   `json:"minimizedAfterMinimize"`
		MinimizedAfterRestore     bool   `json:"minimizedAfterRestore"`
		MaximizedAfterMaximize    bool   `json:"maximizedAfterMaximize"`
		MaximizedAfterUnmaximize  bool   `json:"maximizedAfterUnmaximize"`
		ClosePrevented            bool   `json:"closePrevented"`
		UsableAfterPreventedClose bool   `json:"usableAfterPreventedClose"`
		CloseEvent                bool   `json:"closeEvent"`
		ClosedEvent               bool   `json:"closedEvent"`
		Closed                    bool   `json:"closed"`
		EnabledAfterEnd           bool   `json:"enabledAfterEnd"`
		Error                     string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--browser-window-methods-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("browser window methods error = %q", payload.Error)
	}
	if payload.InitialVisible || !payload.AfterShow || payload.AfterHide {
		t.Fatalf("visibility transitions = initial %v show %v hide %v, want false/true/false", payload.InitialVisible, payload.AfterShow, payload.AfterHide)
	}
	if payload.Bounds.X != 10 || payload.Bounds.Y != 20 || payload.Bounds.Width != 320 || payload.Bounds.Height != 240 {
		t.Fatalf("bounds = %#v, want 10,20 320x240", payload.Bounds)
	}
	if payload.NormalBounds != payload.Bounds {
		t.Fatalf("normalBounds = %#v, want bounds %#v", payload.NormalBounds, payload.Bounds)
	}
	wantEvents := []string{"show", "minimize", "restore", "maximize", "unmaximize", "close-prevented", "close", "closed"}
	if !reflect.DeepEqual(payload.EventOrder, wantEvents) {
		t.Fatalf("eventOrder = %#v, want %#v", payload.EventOrder, wantEvents)
	}
	if payload.EnabledAfterDisable || !payload.EnabledAfterEnable {
		t.Fatalf("enabled transitions = disable %v enable %v, want false/true", payload.EnabledAfterDisable, payload.EnabledAfterEnable)
	}
	if !payload.MinimizedAfterMinimize || payload.MinimizedAfterRestore {
		t.Fatalf("minimize transitions = minimize %v restore %v, want true/false", payload.MinimizedAfterMinimize, payload.MinimizedAfterRestore)
	}
	if !payload.MaximizedAfterMaximize || payload.MaximizedAfterUnmaximize {
		t.Fatalf("maximize transitions = maximize %v unmaximize %v, want true/false", payload.MaximizedAfterMaximize, payload.MaximizedAfterUnmaximize)
	}
	if !payload.ClosePrevented || !payload.UsableAfterPreventedClose {
		t.Fatalf("prevented close state = prevented %v usable %v, want true/true", payload.ClosePrevented, payload.UsableAfterPreventedClose)
	}
	if !payload.CloseEvent || !payload.ClosedEvent || !payload.Closed || payload.EnabledAfterEnd {
		t.Fatalf("close state = closeEvent %v closedEvent %v closed %v enabledAfterEnd %v, want true/true/true/false", payload.CloseEvent, payload.ClosedEvent, payload.Closed, payload.EnabledAfterEnd)
	}
}

func TestRunViewTreeCheckReportsChildOrdering(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--view-tree-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--view-tree-check) exit = %d, want 0", code)
	}

	var payload struct {
		BoundsChanged           bool   `json:"boundsChanged"`
		RootChildWidths         []int  `json:"rootChildWidths"`
		AfterRemoveWidths       []int  `json:"afterRemoveWidths"`
		AfterReparentWidths     []int  `json:"afterReparentWidths"`
		NewParentChildWidths    []int  `json:"newParentChildWidths"`
		LiveWebContentsLoaded   bool   `json:"liveWebContentsLoaded"`
		LiveWebContentsURL      string `json:"liveWebContentsURL"`
		LiveWebContentsParented bool   `json:"liveWebContentsParented"`
		LiveWebContentsDetached bool   `json:"liveWebContentsDetached"`
		ProbeBounds             struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"probeBounds"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--view-tree-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("view tree error = %q", payload.Error)
	}
	if !payload.BoundsChanged {
		t.Fatal("boundsChanged = false, want true")
	}
	if !reflect.DeepEqual(payload.RootChildWidths, []int{100, 300, 200}) {
		t.Fatalf("rootChildWidths = %#v, want [100 300 200]", payload.RootChildWidths)
	}
	if !reflect.DeepEqual(payload.AfterRemoveWidths, []int{100, 300}) {
		t.Fatalf("afterRemoveWidths = %#v, want [100 300]", payload.AfterRemoveWidths)
	}
	if !reflect.DeepEqual(payload.AfterReparentWidths, []int{300}) {
		t.Fatalf("afterReparentWidths = %#v, want [300]", payload.AfterReparentWidths)
	}
	if !reflect.DeepEqual(payload.NewParentChildWidths, []int{100}) {
		t.Fatalf("newParentChildWidths = %#v, want [100]", payload.NewParentChildWidths)
	}
	if payload.ProbeBounds.X != 7 || payload.ProbeBounds.Y != 8 || payload.ProbeBounds.Width != 300 || payload.ProbeBounds.Height != 200 {
		t.Fatalf("probeBounds = %#v, want 7,8 300x200", payload.ProbeBounds)
	}
	if !payload.LiveWebContentsParented || !payload.LiveWebContentsLoaded || payload.LiveWebContentsURL != "data:" || !payload.LiveWebContentsDetached {
		t.Fatalf("live WebContentsView state = %#v, want parented loaded data: detached", payload)
	}
}

func TestRunViewAnimationCheckReportsAnimatedBounds(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--view-animation-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--view-animation-check) exit = %d, want 0", code)
	}

	var payload struct {
		Animated                      bool   `json:"animated"`
		DurationMS                    int    `json:"durationMS"`
		Easing                        string `json:"easing"`
		BoundsChanged                 bool   `json:"boundsChanged"`
		BackgroundBlurMethodAvailable bool   `json:"backgroundBlurMethodAvailable"`
		BackgroundBlurAccepted        bool   `json:"backgroundBlurAccepted"`
		Bounds                        struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"bounds"`
		Error string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--view-animation-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("view animation error = %q", payload.Error)
	}
	if !payload.Animated || payload.DurationMS != 300 || payload.Easing != "ease-in-out" || !payload.BoundsChanged {
		t.Fatalf("animation report = %#v, want animated duration 300 easing ease-in-out boundsChanged", payload)
	}
	if payload.BackgroundBlurMethodAvailable || payload.BackgroundBlurAccepted {
		t.Fatalf("background blur report = %#v, want unavailable and unaccepted for observed Electron 42 View API", payload)
	}
	if payload.Bounds.X != 0 || payload.Bounds.Y != 0 || payload.Bounds.Width != 300 || payload.Bounds.Height != 200 {
		t.Fatalf("bounds = %#v, want 0,0 300x200", payload.Bounds)
	}
}

func TestRunWebContentsNavigationCheckReportsNavigationState(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--webcontents-navigation-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--webcontents-navigation-check) exit = %d, want 0", code)
	}

	var payload struct {
		DidStartNavigation bool   `json:"didStartNavigation"`
		DidNavigate        bool   `json:"didNavigate"`
		DidFinishLoad      bool   `json:"didFinishLoad"`
		DidNavigateInPage  bool   `json:"didNavigateInPage"`
		CanGoBack          bool   `json:"canGoBack"`
		CanGoForward       bool   `json:"canGoForward"`
		LoadingAfter       bool   `json:"loadingAfter"`
		FinalHash          string `json:"finalHash"`
		HistoryLength      int    `json:"historyLength"`
		Error              string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--webcontents-navigation-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("webContents navigation error = %q", payload.Error)
	}
	if !payload.DidStartNavigation || !payload.DidNavigate || !payload.DidFinishLoad || payload.DidNavigateInPage {
		t.Fatalf("navigation event flags = %#v, want document navigation without in-page navigation", payload)
	}
	if payload.CanGoBack || payload.CanGoForward || payload.LoadingAfter {
		t.Fatalf("history/loading state = back %v forward %v loading %v, want false/false/false", payload.CanGoBack, payload.CanGoForward, payload.LoadingAfter)
	}
	if payload.FinalHash != "" || payload.HistoryLength != 1 {
		t.Fatalf("final hash/history = %q/%d, want empty/1", payload.FinalHash, payload.HistoryLength)
	}
}

func TestRunWebContentsDevToolsTargetCheckReportsStableIDs(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--webcontents-devtools-target-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--webcontents-devtools-target-check) exit = %d, want 0", code)
	}

	var payload struct {
		MethodAvailable bool   `json:"methodAvailable"`
		FirstNonempty   bool   `json:"firstNonempty"`
		Stable          bool   `json:"stable"`
		Distinct        bool   `json:"distinct"`
		Lookupable      bool   `json:"lookupable"`
		Error           string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--webcontents-devtools-target-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("webContents DevTools target error = %q", payload.Error)
	}
	if !payload.MethodAvailable || !payload.FirstNonempty || !payload.Stable || !payload.Distinct || !payload.Lookupable {
		t.Fatalf("DevTools target report = %#v, want all true", payload)
	}
}

func TestRunWebContentsPrintCheckReportsDefaultPageSizeContract(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--webcontents-print-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--webcontents-print-check) exit = %d, want 0", code)
	}

	var payload struct {
		DefaultAccepted  bool   `json:"defaultAccepted"`
		DefaultPageSize  bool   `json:"defaultPageSize"`
		PageSizeOmitted  bool   `json:"pageSizeOmitted"`
		CopiesDefault    int    `json:"copiesDefault"`
		ConflictRejected bool   `json:"conflictRejected"`
		Error            string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--webcontents-print-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("webContents print error = %q", payload.Error)
	}
	if !payload.DefaultAccepted || !payload.DefaultPageSize || !payload.PageSizeOmitted || payload.CopiesDefault != 1 || !payload.ConflictRejected {
		t.Fatalf("print report = %#v, want accepted default page size, no pageSize, copies 1, conflict rejected", payload)
	}
}

func TestRunFocusOnNavigationCheckReportsDefaultAndExplicitFalse(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--focus-on-navigation-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--focus-on-navigation-check) exit = %d, want 0", code)
	}

	var payload struct {
		DefaultAccepted       bool   `json:"defaultAccepted"`
		DefaultFocuses        bool   `json:"defaultFocuses"`
		ExplicitFalseAccepted bool   `json:"explicitFalseAccepted"`
		ExplicitFalseFocuses  bool   `json:"explicitFalseFocuses"`
		Error                 string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--focus-on-navigation-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("focusOnNavigation error = %q", payload.Error)
	}
	if !payload.DefaultAccepted || !payload.DefaultFocuses || !payload.ExplicitFalseAccepted || payload.ExplicitFalseFocuses {
		t.Fatalf("focusOnNavigation report = %#v, want accepted true default and accepted false explicit", payload)
	}
}

func TestRunOffscreenDeviceScaleCheckReportsDefaultAndCustomScale(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--offscreen-device-scale-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--offscreen-device-scale-check) exit = %d, want 0", code)
	}

	var payload struct {
		DefaultAccepted          bool    `json:"defaultAccepted"`
		DefaultDeviceScaleFactor float64 `json:"defaultDeviceScaleFactor"`
		CustomAccepted           bool    `json:"customAccepted"`
		CustomDeviceScaleFactor  float64 `json:"customDeviceScaleFactor"`
		Error                    string  `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--offscreen-device-scale-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("offscreen device scale error = %q", payload.Error)
	}
	if !payload.DefaultAccepted || payload.DefaultDeviceScaleFactor != 1 || !payload.CustomAccepted || payload.CustomDeviceScaleFactor != 2 {
		t.Fatalf("offscreen report = %#v, want accepted default scale 1 and custom scale 2", payload)
	}
}

func TestRunChromiumFeaturesCheckReportsCapabilities(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--chromium-features-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--chromium-features-check) exit = %d, want 0", code)
	}

	var payload struct {
		ChromiumVersion        string   `json:"chromiumVersion"`
		WebGL                  bool     `json:"webgl"`
		WebGPU                 bool     `json:"webgpu"`
		PDF                    bool     `json:"pdf"`
		MediaCapture           bool     `json:"mediaCapture"`
		SharedTextures         []string `json:"sharedTextures"`
		LOAFAttribution        bool     `json:"loafAttribution"`
		WasmTrapHandler        bool     `json:"wasmTrapHandler"`
		FeatureFlagWorks       bool     `json:"featureFlagWorks"`
		DiagnosticRecorded     bool     `json:"diagnosticRecorded"`
		DiagnosticCopyIsolated bool     `json:"diagnosticCopyIsolated"`
		Error                  string   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--chromium-features-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("chromium features error = %q", payload.Error)
	}
	if payload.ChromiumVersion != "148.0.7778.96" || !payload.WebGL || !payload.WebGPU || !payload.PDF || !payload.MediaCapture {
		t.Fatalf("chromium capability report = %#v", payload)
	}
	if len(payload.SharedTextures) != 2 || payload.SharedTextures[0] != "rgba8" || payload.SharedTextures[1] != "rgb10a2" {
		t.Fatalf("shared textures = %#v, want rgba8/rgb10a2", payload.SharedTextures)
	}
	if !payload.LOAFAttribution || !payload.WasmTrapHandler || !payload.FeatureFlagWorks || !payload.DiagnosticRecorded || !payload.DiagnosticCopyIsolated {
		t.Fatalf("chromium feature flags/diagnostics = %#v", payload)
	}
}

func TestRunIPCCheckReportsInvokeAndHandlerContract(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--ipc-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--ipc-check) exit = %d, want 0", code)
	}

	var payload struct {
		InvokePong             bool   `json:"invokePong"`
		ArgsEcho               bool   `json:"argsEcho"`
		OnceFirst              bool   `json:"onceFirst"`
		OnceSecondRejected     bool   `json:"onceSecondRejected"`
		RemovedRejected        bool   `json:"removedRejected"`
		DuplicateRejected      bool   `json:"duplicateRejected"`
		MissingRejected        bool   `json:"missingRejected"`
		MessagePortRoundTrip   bool   `json:"messagePortRoundTrip"`
		TransferredPortMessage bool   `json:"transferredPortMessage"`
		Error                  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--ipc-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("ipc error = %q", payload.Error)
	}
	if !payload.InvokePong || !payload.ArgsEcho || !payload.OnceFirst || !payload.OnceSecondRejected || !payload.RemovedRejected || !payload.DuplicateRejected || !payload.MissingRejected || !payload.MessagePortRoundTrip || !payload.TransferredPortMessage {
		t.Fatalf("ipc report = %#v, want all contract flags true", payload)
	}
}

func TestRunContextBridgeCheckReportsPreloadExposure(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--context-bridge-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--context-bridge-check) exit = %d, want 0", code)
	}

	var payload struct {
		ExposedVersion    bool   `json:"exposedVersion"`
		FunctionCallable  bool   `json:"functionCallable"`
		ArrayValueCopied  bool   `json:"arrayValueCopied"`
		DuplicateRejected bool   `json:"duplicateRejected"`
		MutationRejected  bool   `json:"mutationRejected"`
		Error             string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--context-bridge-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("contextBridge error = %q", payload.Error)
	}
	if !payload.ExposedVersion || !payload.FunctionCallable || !payload.ArrayValueCopied || !payload.DuplicateRejected || !payload.MutationRejected {
		t.Fatalf("contextBridge report = %#v, want all contract flags true", payload)
	}
}

func TestRunHelloUsesFixture(t *testing.T) {
	code := run([]string{"electron-go", "--hello"}, nil)
	if code != 78 {
		t.Fatalf("run(--hello) exit = %d, want bridge-unavailable 78", code)
	}
}

func TestRunRejectsUnknownFlag(t *testing.T) {
	code := run([]string{"electron-go", "--not-a-real-flag"}, nil)
	if code != 2 {
		t.Fatalf("run(unknown flag) exit = %d, want 2", code)
	}
}

func TestRunPositionalAppLaunchUsesFastPath(t *testing.T) {
	const wantAppDir = "compat/fixtures/benchmark-hello"
	const wantCode = 17
	var called bool
	restore := replaceLaunchAppForRun(t, func(ctx context.Context, appDir string, argv []string, env []string, nodeOptions egruntime.NodeOptions) int {
		called = true
		if appDir != wantAppDir {
			t.Fatalf("appDir = %q, want %q", appDir, wantAppDir)
		}
		if !reflect.DeepEqual(argv, []string{"electron-go", wantAppDir}) {
			t.Fatalf("argv = %#v", argv)
		}
		if !reflect.DeepEqual(env, []string{"ELECTRON_GO_TEST=1"}) {
			t.Fatalf("env = %#v", env)
		}
		if nodeOptions.ExperimentalTransformTypes {
			t.Fatal("ExperimentalTransformTypes = true, want false for fast-path app launch")
		}
		return wantCode
	})
	defer restore()

	code := run([]string{"electron-go", wantAppDir}, []string{"ELECTRON_GO_TEST=1"})
	if code != wantCode {
		t.Fatalf("run(positional app) exit = %d, want %d", code, wantCode)
	}
	if !called {
		t.Fatal("launchAppForRun was not called")
	}
}

func TestRunLeadingFlagUsesCompatibilityCLI(t *testing.T) {
	restore := replaceLaunchAppForRun(t, func(ctx context.Context, appDir string, argv []string, env []string, nodeOptions egruntime.NodeOptions) int {
		t.Fatalf("launchAppForRun called for leading diagnostic flag: appDir=%q argv=%#v", appDir, argv)
		return 99
	})
	defer restore()

	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--check-parity"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--check-parity) exit = %d, want 0", code)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("parity complete")) {
		t.Fatalf("stdout = %q, want parity complete", stdout.String())
	}
}

func TestShouldFastPathAppLaunch(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "no args launches current directory", want: true},
		{name: "positional app directory", args: []string{"compat/fixtures/benchmark-hello"}, want: true},
		{name: "flags use compatibility CLI", args: []string{"--check-parity"}, want: false},
		{name: "dash separator uses compatibility CLI parsing", args: []string{"--", "compat/fixtures/benchmark-hello"}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := shouldFastPathAppLaunch(test.args); got != test.want {
				t.Fatalf("shouldFastPathAppLaunch(%v) = %v, want %v", test.args, got, test.want)
			}
		})
	}
}

func TestRunAcceptsExperimentalTransformTypes(t *testing.T) {
	code := run([]string{"electron-go", "--experimental-transform-types", "--hello"}, nil)
	if code != 78 {
		t.Fatalf("run(--experimental-transform-types --hello) exit = %d, want bridge-unavailable 78", code)
	}
}

func TestRunMainSkipsEnvironmentForCEFSubprocess(t *testing.T) {
	called := false
	code := runMain([]string{"electron-go", "--type=renderer"}, func() []string {
		called = true
		return []string{"ELECTRON_GO_TEST=1"}
	})
	if code != 78 {
		t.Fatalf("runMain(CEF subprocess) exit = %d, want bridge-unavailable 78", code)
	}
	if called {
		t.Fatal("environment provider called for CEF subprocess")
	}
}

func replaceLaunchAppForRun(t *testing.T, launcher func(context.Context, string, []string, []string, egruntime.NodeOptions) int) func() {
	t.Helper()
	old := launchAppForRun
	launchAppForRun = launcher
	return func() {
		launchAppForRun = old
	}
}

func TestRunAppPathsCheckReportsPathRelationships(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--app-paths-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--app-paths-check) exit = %d, want 0", code)
	}

	var payload struct {
		AppPathNonempty             bool   `json:"appPathNonempty"`
		HomePathNonempty            bool   `json:"homePathNonempty"`
		AppDataPathNonempty         bool   `json:"appDataPathNonempty"`
		UserDataPathNonempty        bool   `json:"userDataPathNonempty"`
		TempPathNonempty            bool   `json:"tempPathNonempty"`
		ExePathNonempty             bool   `json:"exePathNonempty"`
		ModulePathNonempty          bool   `json:"modulePathNonempty"`
		DesktopPathNonempty         bool   `json:"desktopPathNonempty"`
		DocumentsPathNonempty       bool   `json:"documentsPathNonempty"`
		DownloadsPathNonempty       bool   `json:"downloadsPathNonempty"`
		MusicPathNonempty           bool   `json:"musicPathNonempty"`
		PicturesPathNonempty        bool   `json:"picturesPathNonempty"`
		VideosPathNonempty          bool   `json:"videosPathNonempty"`
		CrashDumpsPathNonempty      bool   `json:"crashDumpsPathNonempty"`
		SessionDataDefaultsUserData bool   `json:"sessionDataDefaultsUserData"`
		SessionDataOverrideWorks    bool   `json:"sessionDataOverrideWorks"`
		LogsOverrideWorks           bool   `json:"logsOverrideWorks"`
		Error                       string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--app-paths-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("app paths error = %q", payload.Error)
	}
	if !payload.AppPathNonempty || !payload.HomePathNonempty || !payload.AppDataPathNonempty || !payload.UserDataPathNonempty {
		t.Fatalf("app path nonempty flags are wrong: %#v", payload)
	}
	if !payload.TempPathNonempty || !payload.ExePathNonempty || !payload.ModulePathNonempty || !payload.DesktopPathNonempty || !payload.DocumentsPathNonempty || !payload.DownloadsPathNonempty || !payload.MusicPathNonempty || !payload.PicturesPathNonempty || !payload.VideosPathNonempty || !payload.CrashDumpsPathNonempty {
		t.Fatalf("expanded app path nonempty flags are wrong: %#v", payload)
	}
	if !payload.SessionDataDefaultsUserData || !payload.SessionDataOverrideWorks || !payload.LogsOverrideWorks {
		t.Fatalf("app path relationship flags are wrong: %#v", payload)
	}
}

func TestRunAsarCheckReportsArchiveReads(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--asar-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--asar-check) exit = %d, want 0", code)
	}

	var payload struct {
		ReadMain           bool   `json:"readMain"`
		RequireIndex       bool   `json:"requireIndex"`
		UnpackedRead       bool   `json:"unpackedRead"`
		CopySourceUnpacked bool   `json:"copySourceUnpacked"`
		StatSize           int64  `json:"statSize"`
		Error              string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--asar-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("asar error = %q", payload.Error)
	}
	if !payload.ReadMain || !payload.RequireIndex || !payload.UnpackedRead || payload.CopySourceUnpacked || payload.StatSize == 0 {
		t.Fatalf("asar report = %#v, want packed reads, module resolution, no unpacked source, and stat size", payload)
	}
}

func TestRunAppWebAuthnCheckReportsTouchIDConfig(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--app-webauthn-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--app-webauthn-check) exit = %d, want 0", code)
	}

	var payload struct {
		Configured          bool   `json:"configured"`
		TouchIDSupported    bool   `json:"touchIDSupported"`
		KeychainGroupStored bool   `json:"keychainGroupStored"`
		InvalidRejected     bool   `json:"invalidRejected"`
		Error               string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--app-webauthn-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("app WebAuthn error = %q", payload.Error)
	}
	if !payload.Configured || !payload.KeychainGroupStored {
		t.Fatalf("app WebAuthn config flags = %#v, want configured and stored", payload)
	}
	if goruntime.GOOS == "darwin" && !payload.TouchIDSupported {
		t.Fatalf("touchIDSupported = false on darwin: %#v", payload)
	}
	if payload.InvalidRejected {
		t.Fatalf("invalidRejected = true, want false for observed Electron app.configureWebAuthn behavior")
	}
}

func TestRunGlobalShortcutCheckReportsRegistrationLifecycle(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--global-shortcut-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--global-shortcut-check) exit = %d, want 0", code)
	}

	var payload struct {
		Registered           bool   `json:"registered"`
		DuplicateRejected    bool   `json:"duplicateRejected"`
		AliasRegistered      bool   `json:"aliasRegistered"`
		Unregistered         bool   `json:"unregistered"`
		UnregisterAllCleared bool   `json:"unregisterAllCleared"`
		Error                string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--global-shortcut-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("globalShortcut error = %q", payload.Error)
	}
	if !payload.Registered || !payload.DuplicateRejected || !payload.AliasRegistered || !payload.Unregistered || !payload.UnregisterAllCleared {
		t.Fatalf("globalShortcut report = %#v, want registration lifecycle flags true", payload)
	}
}

func TestRunAppLifecycleCheckReportsReadyAndQuitOrder(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--app-lifecycle-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--app-lifecycle-check) exit = %d, want 0", code)
	}

	var payload struct {
		Ready               bool     `json:"ready"`
		WhenReady           bool     `json:"whenReady"`
		EventOrder          []string `json:"eventOrder"`
		CancelledBeforeQuit bool     `json:"cancelledBeforeQuit"`
		QuitExitCode        int      `json:"quitExitCode"`
		Error               string   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--app-lifecycle-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("app lifecycle error = %q", payload.Error)
	}
	if !payload.Ready || !payload.WhenReady {
		t.Fatalf("ready flags = ready %v whenReady %v, want true/true", payload.Ready, payload.WhenReady)
	}
	if !payload.CancelledBeforeQuit {
		t.Fatalf("cancelledBeforeQuit = false, want true")
	}
	want := []string{"when-ready", "ready", "before-quit-cancelled", "before-quit", "will-quit", "quit"}
	if !reflect.DeepEqual(payload.EventOrder, want) {
		t.Fatalf("eventOrder = %#v, want %#v", payload.EventOrder, want)
	}
	if payload.QuitExitCode != 0 {
		t.Fatalf("quitExitCode = %d, want 0", payload.QuitExitCode)
	}
}

func TestRunNodeOptionsCheckReportsExperimentalTransformTypes(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--experimental-transform-types", "--node-options-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--experimental-transform-types --node-options-check) exit = %d, want 0", code)
	}

	var payload struct {
		Supported                  bool   `json:"supported"`
		ExperimentalTransformTypes bool   `json:"experimentalTransformTypes"`
		ExecArgvIncludesFlag       bool   `json:"execArgvIncludesFlag"`
		Node                       string `json:"node"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--node-options-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if !payload.Supported {
		t.Fatal("supported = false, want true")
	}
	if !payload.ExperimentalTransformTypes {
		t.Fatal("experimentalTransformTypes = false, want true")
	}
	if payload.ExecArgvIncludesFlag {
		t.Fatal("execArgvIncludesFlag = true, want false for Electron app launch")
	}
	if payload.Node == "" {
		t.Fatal("node version is empty")
	}
}

func TestRunNodeVersionsCheckReportsTargetVersions(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--node-versions-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--node-versions-check) exit = %d, want 0", code)
	}

	var payload struct {
		Electron          string `json:"electron"`
		Chrome            string `json:"chrome"`
		Node              string `json:"node"`
		V8                string `json:"v8"`
		Modules           string `json:"modules"`
		CJSRequireWorks   bool   `json:"cjsRequireWorks"`
		ESMImportWorks    bool   `json:"esmImportWorks"`
		NativeABIReported bool   `json:"nativeABIReported"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--node-versions-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Electron == "" || payload.Chrome == "" || payload.Node == "" || payload.V8 == "" || payload.Modules == "" {
		t.Fatalf("version payload has empty fields: %#v", payload)
	}
	if payload.Node != "24.15.0" {
		t.Fatalf("node = %q, want 24.15.0", payload.Node)
	}
	if payload.V8 != "14.8.178.14-electron.0" || payload.Modules != "146" {
		t.Fatalf("v8/modules = %q/%q, want 14.8.178.14-electron.0/146", payload.V8, payload.Modules)
	}
	if !payload.CJSRequireWorks || !payload.ESMImportWorks || !payload.NativeABIReported {
		t.Fatalf("node module flags = cjs %v esm %v nativeABI %v, want true/true/true", payload.CJSRequireWorks, payload.ESMImportWorks, payload.NativeABIReported)
	}
}

func TestRunNetLogCheckReportsLifecycle(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--netlog-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--netlog-check) exit = %d, want 0", code)
	}

	var payload struct {
		Started       bool   `json:"started"`
		ActiveDuring  bool   `json:"activeDuring"`
		Stopped       bool   `json:"stopped"`
		InactiveAfter bool   `json:"inactiveAfter"`
		Error         string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--netlog-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("netLog error = %q", payload.Error)
	}
	if !payload.Started || !payload.ActiveDuring || !payload.Stopped || !payload.InactiveAfter {
		t.Fatalf("netLog lifecycle = %#v, want all true", payload)
	}
}

func TestRunClientRequestCheckReportsNormalizedRequest(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--client-request-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--client-request-check) exit = %d, want 0", code)
	}

	var payload struct {
		Method         string `json:"method"`
		Path           string `json:"path"`
		QueryMode      string `json:"queryMode"`
		Header         string `json:"header"`
		Body           string `json:"body"`
		RedirectPolicy string `json:"redirectPolicy"`
		Ended          bool   `json:"ended"`
		Error          string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--client-request-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("client request error = %q", payload.Error)
	}
	if payload.Method != "POST" || payload.Path != "/request" || payload.QueryMode != "manual" || payload.Header != "one" || payload.Body != "payload" || payload.RedirectPolicy != "manual" || !payload.Ended {
		t.Fatalf("client request report = %#v, want POST /request?mode=manual one/payload/manual/ended", payload)
	}
}

func TestRunNotificationCheckReportsConstructorOptions(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--notification-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--notification-check) exit = %d, want 0", code)
	}

	var payload struct {
		Title          string `json:"title"`
		Body           string `json:"body"`
		Silent         bool   `json:"silent"`
		DefaultUrgency string `json:"defaultUrgency"`
		Error          string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--notification-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("notification error = %q", payload.Error)
	}
	if payload.Title != "Build complete" || payload.Body != "Artifacts are ready" || !payload.Silent {
		t.Fatalf("notification report = %#v, want constructor options", payload)
	}
	if payload.DefaultUrgency != "normal" {
		t.Fatalf("defaultUrgency = %q, want normal", payload.DefaultUrgency)
	}
}

func TestRunNotificationIdentityCheckReportsPlatformSurface(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--notification-identity-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--notification-identity-check) exit = %d, want 0", code)
	}

	var payload struct {
		Platform                   string `json:"platform"`
		Supported                  bool   `json:"supported"`
		HistoryAvailable           bool   `json:"historyAvailable"`
		RemoveFromHistoryAvailable bool   `json:"removeFromHistoryAvailable"`
		ID                         string `json:"id,omitempty"`
		GroupID                    string `json:"groupId,omitempty"`
		GroupTitle                 string `json:"groupTitle,omitempty"`
		Title                      string `json:"title,omitempty"`
		Body                       string `json:"body,omitempty"`
		Error                      string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--notification-identity-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("notification identity error = %q", payload.Error)
	}
	if payload.Platform == "" {
		t.Fatalf("notification identity platform is empty: %#v", payload)
	}
	if payload.Platform == "darwin" && (!payload.HistoryAvailable || payload.RemoveFromHistoryAvailable) {
		t.Fatalf("notification history API flags = %#v, want getHistory available and removeFromHistory unavailable on darwin", payload)
	}
	if payload.Supported {
		if payload.ID != "deploy-42" || payload.GroupID != "deployments" || payload.Title != "Deploy complete" || payload.Body != "Artifacts are ready" {
			t.Fatalf("notification identity report = %#v, want id/group/title/body fields", payload)
		}
	}
}

func TestRunGlobalShortcutSuspensionCheckReportsStateTransitions(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--global-shortcut-suspension-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--global-shortcut-suspension-check) exit = %d, want 0", code)
	}

	var payload struct {
		InitialSuspended  bool   `json:"initialSuspended"`
		SuspendedAfterSet bool   `json:"suspendedAfterSet"`
		ResumedAfterUnset bool   `json:"resumedAfterUnset"`
		Error             string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--global-shortcut-suspension-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("globalShortcut suspension error = %q", payload.Error)
	}
	if payload.InitialSuspended || !payload.SuspendedAfterSet || !payload.ResumedAfterUnset {
		t.Fatalf("globalShortcut suspension state = %#v, want false/true/true", payload)
	}
}

func TestRunAutoUpdaterCheckReportsFeedURL(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--auto-updater-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--auto-updater-check) exit = %d, want 0", code)
	}

	var payload struct {
		Platform        string `json:"platform"`
		Supported       bool   `json:"supported"`
		FeedURL         string `json:"feedURL"`
		DefaultProvider string `json:"defaultProvider,omitempty"`
		Error           string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--auto-updater-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("autoUpdater error = %q", payload.Error)
	}
	if payload.Platform == "" {
		t.Fatalf("autoUpdater platform is empty: %#v", payload)
	}
	if payload.Supported {
		if payload.FeedURL != "https://updates.example.test/feed" || payload.DefaultProvider != "squirrel" {
			t.Fatalf("autoUpdater report = %#v, want feed URL and squirrel provider", payload)
		}
	}
}

func TestRunMenuCheckReportsTemplateNormalization(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--menu-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--menu-check) exit = %d, want 0", code)
	}

	var payload struct {
		ItemCount                int    `json:"itemCount"`
		FirstLabel               string `json:"firstLabel"`
		FirstEnabled             bool   `json:"firstEnabled"`
		CheckboxLabel            string `json:"checkboxLabel"`
		CheckboxChecked          bool   `json:"checkboxChecked"`
		SubmenuFound             bool   `json:"submenuFound"`
		SubmenuRole              string `json:"submenuRole"`
		ApplicationMenuSet       bool   `json:"applicationMenuSet"`
		ApplicationMenuRetrieved bool   `json:"applicationMenuRetrieved"`
		TrayCreated              bool   `json:"trayCreated"`
		TrayTitleSet             bool   `json:"trayTitleSet"`
		TrayToolTipSet           bool   `json:"trayToolTipSet"`
		TrayContextMenuSet       bool   `json:"trayContextMenuSet"`
		DynamicLabelUpdated      bool   `json:"dynamicLabelUpdated"`
		DynamicEnabledUpdated    bool   `json:"dynamicEnabledUpdated"`
		Error                    string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--menu-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("menu error = %q", payload.Error)
	}
	if payload.ItemCount != 4 || payload.FirstLabel != " Open " || !payload.FirstEnabled {
		t.Fatalf("menu first item report = %#v", payload)
	}
	if payload.CheckboxLabel != "Enabled" || !payload.CheckboxChecked {
		t.Fatalf("menu checkbox report = %#v", payload)
	}
	if !payload.SubmenuFound || payload.SubmenuRole != "toggledevtools" {
		t.Fatalf("menu submenu report = %#v", payload)
	}
	if !payload.DynamicLabelUpdated || !payload.DynamicEnabledUpdated {
		t.Fatalf("dynamic menu update report = label %v enabled %v, want true/true", payload.DynamicLabelUpdated, payload.DynamicEnabledUpdated)
	}
	if !payload.ApplicationMenuSet || !payload.ApplicationMenuRetrieved {
		t.Fatalf("application menu report = set %v retrieved %v, want true/true", payload.ApplicationMenuSet, payload.ApplicationMenuRetrieved)
	}
	if !payload.TrayCreated || !payload.TrayTitleSet || !payload.TrayToolTipSet || !payload.TrayContextMenuSet {
		t.Fatalf("tray report = %#v, want created/title/tooltip/context menu", payload)
	}
}

func TestRunNotificationFailureCheckReportsMacUnsignedFailure(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--notification-failure-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--notification-failure-check) exit = %d, want 0", code)
	}

	var payload struct {
		Platform    string `json:"platform"`
		Unsupported bool   `json:"unsupported"`
		Failed      bool   `json:"failed"`
		Shown       bool   `json:"shown"`
		ErrorDomain bool   `json:"errorDomain"`
		Error       string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--notification-failure-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("notification failure error = %q", payload.Error)
	}
	if payload.Platform == "darwin" && (!payload.Failed || payload.Shown || !payload.ErrorDomain) {
		t.Fatalf("darwin notification failure report = %#v, want failed/no show/error domain", payload)
	}
	if payload.Platform != "darwin" && !payload.Unsupported {
		t.Fatalf("non-darwin notification failure report = %#v, want unsupported", payload)
	}
}

func TestRunNativeThemeCheckReportsSnapshot(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--native-theme-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--native-theme-check) exit = %d, want 0", code)
	}

	var payload struct {
		Platform                          string `json:"platform"`
		SupportsNativeThemeCore           bool   `json:"supportsNativeThemeCore"`
		ShouldDifferentiateWithoutColor   bool   `json:"shouldDifferentiateWithoutColor"`
		SupportsDifferentiateWithoutColor bool   `json:"supportsDifferentiateWithoutColor"`
		ScreenPrimaryDisplayAvailable     bool   `json:"screenPrimaryDisplayAvailable"`
		ScreenScaleFactorPositive         bool   `json:"screenScaleFactorPositive"`
		PowerMonitorIdleStateAvailable    bool   `json:"powerMonitorIdleStateAvailable"`
		PowerMonitorIdleTimeNonNegative   bool   `json:"powerMonitorIdleTimeNonNegative"`
		SystemPreferencesAvailable        bool   `json:"systemPreferencesAvailable"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--native-theme-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Platform == "" {
		t.Fatal("platform is empty")
	}
	if !payload.SupportsNativeThemeCore {
		t.Fatalf("supportsNativeThemeCore = false: %#v", payload)
	}
	if !payload.SupportsDifferentiateWithoutColor && payload.ShouldDifferentiateWithoutColor {
		t.Fatalf("unsupported platform reported shouldDifferentiateWithoutColor=true: %#v", payload)
	}
	if !payload.ScreenPrimaryDisplayAvailable || !payload.ScreenScaleFactorPositive {
		t.Fatalf("screen report = primary %v scale %v, want true/true", payload.ScreenPrimaryDisplayAvailable, payload.ScreenScaleFactorPositive)
	}
	if !payload.PowerMonitorIdleStateAvailable || !payload.PowerMonitorIdleTimeNonNegative {
		t.Fatalf("powerMonitor report = state %v time %v, want true/true", payload.PowerMonitorIdleStateAvailable, payload.PowerMonitorIdleTimeNonNegative)
	}
}

func TestRunSafeStorageCheckReportsRoundTrip(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--safe-storage-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--safe-storage-check) exit = %d, want 0", code)
	}

	var payload struct {
		Available       bool   `json:"available"`
		AsyncAvailable  bool   `json:"asyncAvailable"`
		Backend         string `json:"backend"`
		RoundTrip       bool   `json:"roundTrip"`
		CiphertextBytes int    `json:"ciphertextBytes"`
		Error           string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--safe-storage-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("safeStorage error = %q", payload.Error)
	}
	if !payload.Available || !payload.AsyncAvailable {
		t.Fatalf("availability = sync %v async %v, want true/true", payload.Available, payload.AsyncAvailable)
	}
	if payload.Backend == "" {
		t.Fatal("backend is empty")
	}
	if !payload.RoundTrip {
		t.Fatal("roundTrip = false, want true")
	}
	if payload.CiphertextBytes == 0 {
		t.Fatal("ciphertextBytes = 0, want encrypted payload")
	}
}

func TestRunProtocolCheckReportsRegistrationLifecycle(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--protocol-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--protocol-check) exit = %d, want 0", code)
	}

	var payload struct {
		RegisteredPrivileged   bool   `json:"registeredPrivileged"`
		AllowExtensions        bool   `json:"allowExtensions"`
		HandledAfterRegister   bool   `json:"handledAfterRegister"`
		FetchStatus            int    `json:"fetchStatus"`
		FetchHeader            bool   `json:"fetchHeader"`
		FetchBody              bool   `json:"fetchBody"`
		DuplicateRejected      bool   `json:"duplicateRejected"`
		LatePrivilegedRejected bool   `json:"latePrivilegedRejected"`
		HandledAfterRemove     bool   `json:"handledAfterRemove"`
		Error                  string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--protocol-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("protocol error = %q", payload.Error)
	}
	if !payload.RegisteredPrivileged || !payload.AllowExtensions || !payload.HandledAfterRegister {
		t.Fatalf("protocol lifecycle missing expected true values: %#v", payload)
	}
	if payload.FetchStatus != 201 || !payload.FetchHeader || !payload.FetchBody {
		t.Fatalf("protocol fetch report = status %d header %v body %v, want 201/true/true", payload.FetchStatus, payload.FetchHeader, payload.FetchBody)
	}
	if !payload.DuplicateRejected {
		t.Fatalf("duplicateRejected = false, want true")
	}
	if !payload.LatePrivilegedRejected {
		t.Fatalf("latePrivilegedRejected = false, want true")
	}
	if payload.HandledAfterRemove {
		t.Fatalf("handledAfterRemove = true, want false")
	}
}

func TestRunPackagingCheckReportsManifestValidation(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--packaging-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--packaging-check) exit = %d, want 0", code)
	}

	var payload struct {
		Valid                        bool   `json:"valid"`
		HasHelpers                   bool   `json:"hasHelpers"`
		HasResources                 bool   `json:"hasResources"`
		Signed                       bool   `json:"signed"`
		Notarized                    bool   `json:"notarized"`
		HardenedRuntime              bool   `json:"hardenedRuntime"`
		MSIXRequiresSigning          bool   `json:"msixRequiresSigning"`
		MASRejectedWithoutCompliance bool   `json:"masRejectedWithoutCompliance"`
		InvalidFuseRejected          bool   `json:"invalidFuseRejected"`
		PackageCount                 int    `json:"packageCount"`
		Error                        string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--packaging-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("packaging error = %q", payload.Error)
	}
	if !payload.Valid || !payload.HasHelpers || !payload.HasResources || !payload.Signed || !payload.Notarized || !payload.HardenedRuntime {
		t.Fatalf("packaging summary = %#v", payload)
	}
	if !payload.MSIXRequiresSigning || !payload.MASRejectedWithoutCompliance || !payload.InvalidFuseRejected || payload.PackageCount != 3 {
		t.Fatalf("packaging validation flags = %#v", payload)
	}
}

func TestRunSessionCheckReportsBasicSessionBehavior(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--session-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--session-check) exit = %d, want 0", code)
	}

	var payload struct {
		DefaultSameWithEmpty       bool   `json:"defaultSameWithEmpty"`
		PersistStoragePathNonempty bool   `json:"persistStoragePathNonempty"`
		MemoryStoragePathEmpty     bool   `json:"memoryStoragePathEmpty"`
		PermissionCheckCalled      bool   `json:"permissionCheckCalled"`
		PermissionRequestCalled    bool   `json:"permissionRequestCalled"`
		PermissionRequestAllowed   bool   `json:"permissionRequestAllowed"`
		CookieRoundTrip            bool   `json:"cookieRoundTrip"`
		CookieCount                int    `json:"cookieCount"`
		CookieOverwriteValue       bool   `json:"cookieOverwriteValue"`
		CookieOverwriteChange      bool   `json:"cookieOverwriteChange"`
		CookieRemoveResolved       bool   `json:"cookieRemoveResolved"`
		CookieRemoved              bool   `json:"cookieRemoved"`
		CookieRemoveChange         bool   `json:"cookieRemoveChange"`
		PartitionCookieIsolation   bool   `json:"partitionCookieIsolation"`
		CacheClearResolved         bool   `json:"cacheClearResolved"`
		StorageClearResolved       bool   `json:"storageClearResolved"`
		StorageClearRemovedCookies bool   `json:"storageClearRemovedCookies"`
		Error                      string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--session-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("session error = %q", payload.Error)
	}
	if !payload.DefaultSameWithEmpty || !payload.PersistStoragePathNonempty || !payload.MemoryStoragePathEmpty {
		t.Fatalf("session identity/partition flags are wrong: %#v", payload)
	}
	if !payload.PermissionCheckCalled || !payload.PermissionRequestCalled || !payload.PermissionRequestAllowed {
		t.Fatalf("permission report = check %v request %v allowed %v, want true/true/true", payload.PermissionCheckCalled, payload.PermissionRequestCalled, payload.PermissionRequestAllowed)
	}
	if !payload.CookieRoundTrip || payload.CookieCount != 1 {
		t.Fatalf("cookie report = roundTrip %v count %d, want true/1", payload.CookieRoundTrip, payload.CookieCount)
	}
	if !payload.CookieOverwriteValue || !payload.CookieOverwriteChange {
		t.Fatalf("cookie overwrite report = value %v change %v, want true/true", payload.CookieOverwriteValue, payload.CookieOverwriteChange)
	}
	if !payload.CookieRemoveResolved || !payload.CookieRemoved || !payload.CookieRemoveChange {
		t.Fatalf("cookie remove report = resolved %v removed %v change %v, want true/true/true", payload.CookieRemoveResolved, payload.CookieRemoved, payload.CookieRemoveChange)
	}
	if !payload.PartitionCookieIsolation {
		t.Fatalf("partitionCookieIsolation = false: %#v", payload)
	}
	if !payload.CacheClearResolved || !payload.StorageClearResolved || !payload.StorageClearRemovedCookies {
		t.Fatalf("clear report = cache %v storage %v removedCookies %v, want true/true/true", payload.CacheClearResolved, payload.StorageClearResolved, payload.StorageClearRemovedCookies)
	}
}

func TestRunSessionQuotasCheckReportsIgnoredRemovedOption(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--session-quotas-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--session-quotas-check) exit = %d, want 0", code)
	}

	var payload struct {
		AcceptsQuotas bool   `json:"acceptsQuotas"`
		ErrorName     string `json:"errorName"`
		Error         string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--session-quotas-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("session quotas error = %q", payload.Error)
	}
	if !payload.AcceptsQuotas {
		t.Fatal("acceptsQuotas = false, want true")
	}
	if payload.ErrorName != "" {
		t.Fatalf("errorName = %q, want empty", payload.ErrorName)
	}
}

func TestRunSessionWebAuthnCheckReportsSelection(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--session-webauthn-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--session-webauthn-check) exit = %d, want 0", code)
	}

	var payload struct {
		HandlerCalled      bool   `json:"handlerCalled"`
		RequestID          string `json:"requestId"`
		Origin             string `json:"origin"`
		CredentialCount    int    `json:"credentialCount"`
		SelectedCredential string `json:"selectedCredential"`
		Error              string `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--session-webauthn-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("session WebAuthn error = %q", payload.Error)
	}
	if !payload.HandlerCalled || payload.RequestID != "request" || payload.Origin != "https://example.test" || payload.CredentialCount != 2 || payload.SelectedCredential != "credential-2" {
		t.Fatalf("session WebAuthn report = %#v, want normalized request and selected credential-2", payload)
	}
}

func TestRunUtilityProcessCheckReportsLifecycleAndOptions(t *testing.T) {
	var stdout bytes.Buffer
	code := runWithOutput(t, []string{"electron-go", "--utility-process-check"}, &stdout, nil)
	if code != 0 {
		t.Fatalf("run(--utility-process-check) exit = %d, want 0", code)
	}

	var payload struct {
		SpawnEvent       bool     `json:"spawnEvent"`
		MessageEvent     bool     `json:"messageEvent"`
		ExitEvent        bool     `json:"exitEvent"`
		ExitCode         int      `json:"exitCode"`
		StdioMode        string   `json:"stdioMode"`
		StdoutPiped      bool     `json:"stdoutPiped"`
		StderrPiped      bool     `json:"stderrPiped"`
		StdinPiped       bool     `json:"stdinPiped"`
		StdinWrite       bool     `json:"stdinWrite"`
		Args             []string `json:"args"`
		EnvironmentValue string   `json:"environmentValue"`
		ServiceName      string   `json:"serviceName"`
		Error            string   `json:"error,omitempty"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("--utility-process-check output is not JSON: %v\n%s", err, stdout.String())
	}
	if payload.Error != "" {
		t.Fatalf("utilityProcess error = %q", payload.Error)
	}
	if !payload.SpawnEvent || !payload.MessageEvent || !payload.ExitEvent || payload.ExitCode != 0 {
		t.Fatalf("utilityProcess lifecycle report = %#v, want spawn/message/exit 0", payload)
	}
	if payload.StdioMode != "pipe" || !payload.StdoutPiped || !payload.StderrPiped || payload.StdinPiped || payload.StdinWrite {
		t.Fatalf("utilityProcess stdio report = %#v, want pipe stdout/stderr without stdin", payload)
	}
	if len(payload.Args) != 1 || payload.Args[0] != "--fixture" || payload.EnvironmentValue != "ok" || payload.ServiceName != "fixture-service" {
		t.Fatalf("utilityProcess option report = %#v, want fixture args/env/service", payload)
	}
}

func runWithOutput(t *testing.T, args []string, stdout, stderr *bytes.Buffer) int {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr
	var stdoutRead, stdoutWrite *os.File
	var stderrRead, stderrWrite *os.File
	var stdoutDone, stderrDone chan error

	if stdout != nil {
		stdoutRead, stdoutWrite = newPipe(t)
		os.Stdout = stdoutWrite
		stdoutDone = drainPipe(stdoutRead, stdout)
	}
	if stderr != nil {
		stderrRead, stderrWrite = newPipe(t)
		os.Stderr = stderrWrite
		stderrDone = drainPipe(stderrRead, stderr)
	}

	code := run(args, nil)

	os.Stdout = oldStdout
	os.Stderr = oldStderr
	if stdoutWrite != nil {
		_ = stdoutWrite.Close()
		if err := <-stdoutDone; err != nil {
			t.Fatalf("reading stdout pipe: %v", err)
		}
		_ = stdoutRead.Close()
	}
	if stderrWrite != nil {
		_ = stderrWrite.Close()
		if err := <-stderrDone; err != nil {
			t.Fatalf("reading stderr pipe: %v", err)
		}
		_ = stderrRead.Close()
	}

	return code
}

func drainPipe(read *os.File, dst *bytes.Buffer) chan error {
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(dst, read)
		done <- err
	}()
	return done
}

func newPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	read, write, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	return read, write
}
