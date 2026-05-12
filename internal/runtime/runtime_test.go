package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gibavargas/electron-go/internal/native"
)

func TestRuntimeReportsStubBridgeUnavailable(t *testing.T) {
	dir := fixtureApp(t)
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge:          native.StubBridge{},
	})

	err := rt.Run(context.Background())
	if !errors.Is(err, native.ErrBridgeUnavailable) {
		t.Fatalf("Run() error = %v, want ErrBridgeUnavailable", err)
	}
	if rt.State() != StateStarting {
		t.Fatalf("State() = %s, want %s", rt.State(), StateStarting)
	}
}

func TestRuntimeStartsBridge(t *testing.T) {
	dir := fixtureApp(t)
	args := []string{"electron-go", "--inspect", dir}
	env := []string{"ELECTRON_ENABLE_LOGGING=1", "PATH=/test/bin"}
	nodeOptions := NodeOptions{ExperimentalTransformTypes: true}
	var got native.StartRequest
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(_ context.Context, req native.StartRequest) (*native.StartResult, error) {
			got = req
			return &native.StartResult{PID: 100, WindowCount: 1, Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"}, nil
		}),
		Args:        args,
		Environment: env,
		NodeOptions: nodeOptions,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if rt.State() != StateStopped {
		t.Fatalf("State() = %s, want %s", rt.State(), StateStopped)
	}
	if got.AppName != "fixture" {
		t.Fatalf("AppName = %q, want fixture", got.AppName)
	}
	if got.ABIRevision != native.CurrentABIRevision {
		t.Fatalf("ABIRevision = %d, want %d", got.ABIRevision, native.CurrentABIRevision)
	}
	if !strings.HasSuffix(got.MainPath, "main.js") {
		t.Fatalf("MainPath = %q, want suffix main.js", got.MainPath)
	}
	if !slices.Equal(got.Args, args) {
		t.Fatalf("Args = %#v, want %#v", got.Args, args)
	}
	if !slices.Equal(got.Environment, env) {
		t.Fatalf("Environment = %#v, want %#v", got.Environment, env)
	}
	if !got.NodeOptions.ExperimentalTransformTypes {
		t.Fatal("NodeOptions.ExperimentalTransformTypes = false, want true")
	}
}

func TestRuntimeIsQuietByDefault(t *testing.T) {
	dir := fixtureApp(t)
	var out bytes.Buffer
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
			return &native.StartResult{PID: 100, WindowCount: 1}, nil
		}),
		Out: &out,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want quiet default", out.String())
	}
}

func TestRuntimeVerboseEnvReportsStatus(t *testing.T) {
	dir := fixtureApp(t)
	var out bytes.Buffer
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
			return &native.StartResult{PID: 100, WindowCount: 1, Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"}, nil
		}),
		Environment: []string{"ELECTRON_GO_VERBOSE=1"},
		Out:         &out,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "electron-go: loaded fixture 1.0.0") {
		t.Fatalf("stdout = %q, want loaded status", got)
	}
	if !strings.Contains(got, "electron-go: running pid=100 windows=1") {
		t.Fatalf("stdout = %q, want running status", got)
	}
}

func TestRuntimeStartupTraceStillPrintsWhenQuiet(t *testing.T) {
	dir := fixtureApp(t)
	var out bytes.Buffer
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
			return &native.StartResult{
				PID:            100,
				WindowCount:    1,
				StartupTraceMS: map[string]int64{"cef_initialize": 12},
			}, nil
		}),
		Environment: []string{"ELECTRON_GO_STARTUP_TRACE=1"},
		Out:         &out,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	got := out.String()
	if strings.Contains(got, "electron-go: loaded") || strings.Contains(got, "electron-go: running") {
		t.Fatalf("stdout = %q, want only startup trace", got)
	}
	if !strings.Contains(got, `electron-go-startup-trace: {"cef_initialize":12}`) {
		t.Fatalf("stdout = %q, want startup trace", got)
	}
}

func TestRuntimeUsesMainPlanBridgeForSupportedMain(t *testing.T) {
	dir := scopedMainApp(t)
	bridge := &mainPlanBridgeFake{browserID: 77}
	var out bytes.Buffer
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge:          bridge,
		Environment:     []string{"ELECTRON_GO_BENCHMARK_TRACE=1", "ELECTRON_GO_STARTUP_TRACE=1"},
		Out:             &out,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if bridge.startCalled {
		t.Fatal("Bridge.Start called, want scoped main-plan path")
	}
	if !bridge.initialized || !bridge.shutdown {
		t.Fatalf("initialized/shutdown = %t/%t, want true/true", bridge.initialized, bridge.shutdown)
	}
	if !strings.HasSuffix(bridge.create.URL, "/index.html") || bridge.create.AutoCloseOnLoad {
		t.Fatalf("create request = %#v, want index.html without auto close", bridge.create)
	}
	if bridge.load.URL != "" {
		t.Fatalf("load URL = %q, want direct create without loadURL", bridge.load.URL)
	}
	if !bridge.waited {
		t.Fatal("bridge did not wait for direct-created load")
	}
	if bridge.close.BrowserID != 77 {
		t.Fatalf("close request = %#v, want browser 77", bridge.close)
	}
	if !strings.Contains(out.String(), "benchmark-trace:") {
		t.Fatalf("stdout = %q, want benchmark trace", out.String())
	}
	if !strings.Contains(out.String(), "electron-go-startup-trace:") {
		t.Fatalf("stdout = %q, want startup trace", out.String())
	}
	startupTrace := parseStartupTrace(t, out.String())
	if startupTrace["cef_initialize"] <= 0 || startupTrace["app_ready"] < startupTrace["cef_initialize"] {
		t.Fatalf("startup trace = %#v, want app marks offset after CEF initialization", startupTrace)
	}
	for _, key := range []string{"cef_initialize", "mainrunner_execute", "cef_shutdown", "total_native_start"} {
		if !strings.Contains(out.String(), `"`+key+`"`) {
			t.Fatalf("stdout = %q, want startup trace key %q", out.String(), key)
		}
	}
}

func TestRuntimeCanSkipFinalShutdownForSupportedMain(t *testing.T) {
	dir := scopedMainApp(t)
	bridge := &mainPlanBridgeFake{browserID: 77}
	var out bytes.Buffer
	rt := New(Options{
		AppDir:            dir,
		ElectronVersion:   "42.0.0",
		Bridge:            bridge,
		Environment:       []string{"ELECTRON_GO_STARTUP_TRACE=1"},
		Out:               &out,
		SkipFinalShutdown: true,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bridge.initialized {
		t.Fatal("bridge was not initialized")
	}
	if bridge.shutdown {
		t.Fatal("bridge shutdown called for final-process run")
	}
	startupTrace := parseStartupTrace(t, out.String())
	if startupTrace["cef_shutdown_skipped"] != 1 {
		t.Fatalf("startup trace = %#v, want cef_shutdown_skipped marker", startupTrace)
	}
	if _, ok := startupTrace["cef_shutdown"]; ok {
		t.Fatalf("startup trace = %#v, did not want cef_shutdown duration", startupTrace)
	}
}

func parseStartupTrace(t *testing.T, output string) map[string]int64 {
	t.Helper()
	const prefix = "electron-go-startup-trace: "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			var trace map[string]int64
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, prefix)), &trace); err != nil {
				t.Fatalf("parse startup trace %q: %v", line, err)
			}
			return trace
		}
	}
	t.Fatalf("stdout = %q, missing startup trace", output)
	return nil
}

func TestRuntimeFallsBackForUnsupportedMainPlan(t *testing.T) {
	dir := fixtureApp(t)
	bridge := &mainPlanBridgeFake{browserID: 77}
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge:          bridge,
	})

	if err := rt.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !bridge.startCalled {
		t.Fatal("Bridge.Start was not called for unsupported main script")
	}
	if bridge.initialized || bridge.shutdown || bridge.create.URL != "" || bridge.load.URL != "" || bridge.close.BrowserID != 0 {
		t.Fatalf("scoped bridge path was used for unsupported main script: %#v", bridge)
	}
}

func TestWarmRunReusesInitializedMainPlanBridge(t *testing.T) {
	dir := scopedMainApp(t)
	bridge := &mainPlanBridgeFake{browserID: 77}
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge:          bridge,
		Environment:     []string{"ELECTRON_GO_BENCHMARK_TRACE=1"},
	})

	report, err := rt.WarmRun(context.Background(), 1)
	if err != nil {
		t.Fatalf("WarmRun() error = %v", err)
	}
	if bridge.startCalled {
		t.Fatal("Bridge.Start called, want direct warm main-plan path")
	}
	if !bridge.initialized || !bridge.shutdown {
		t.Fatalf("initialized/shutdown = %t/%t, want true/true", bridge.initialized, bridge.shutdown)
	}
	if report.Iterations != 1 || len(report.Samples) != 1 {
		t.Fatalf("report iterations/samples = %d/%d, want 1/1", report.Iterations, len(report.Samples))
	}
	if report.Summary.Successes != 1 || report.Summary.Failures != 0 {
		t.Fatalf("summary = %#v, want 1 success", report.Summary)
	}
	for _, sample := range report.Samples {
		if sample.ExitCode != 0 {
			t.Fatalf("sample = %#v, want exit 0", sample)
		}
		if _, ok := sample.StartupTraceMS["app_ready"]; !ok {
			t.Fatalf("sample trace = %#v, want app_ready", sample.StartupTraceMS)
		}
	}
}

func TestWarmRunRejectsRepeatedCEFWindowLoops(t *testing.T) {
	dir := scopedMainApp(t)
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge:          &mainPlanBridgeFake{browserID: 77},
	})

	_, err := rt.WarmRun(context.Background(), 2)
	if err == nil || !strings.Contains(err.Error(), "one visible window") {
		t.Fatalf("WarmRun(iterations=2) error = %v, want clear unsupported repeated-loop error", err)
	}
}

func TestExtractNodeOptions(t *testing.T) {
	if got := ExtractNodeOptions([]string{"electron-go", "--experimental-transform-types", "."}); !got.ExperimentalTransformTypes {
		t.Fatal("ExperimentalTransformTypes = false, want true")
	}
	if got := ExtractNodeOptions([]string{"electron-go", "--inspect", "."}); got.ExperimentalTransformTypes {
		t.Fatal("ExperimentalTransformTypes = true, want false")
	}
}

func TestEnvEnabled(t *testing.T) {
	if !envEnabled([]string{"ELECTRON_GO_STARTUP_TRACE=1"}, StartupTraceEnv) {
		t.Fatal("envEnabled() = false, want true")
	}
	if !envEnabled([]string{"ELECTRON_GO_VERBOSE=1"}, VerboseEnv) {
		t.Fatal("envEnabled() = false for verbose, want true")
	}
	if envEnabled([]string{"ELECTRON_GO_STARTUP_TRACE=0"}, StartupTraceEnv) {
		t.Fatal("envEnabled() = true for 0, want false")
	}
	if envEnabled([]string{"ELECTRON_GO_STARTUP_TRACE=false"}, StartupTraceEnv) {
		t.Fatal("envEnabled() = true for false, want false")
	}
}

func TestRuntimeSubprocessHookShortCircuitsBeforeAppInit(t *testing.T) {
	args := []string{"electron-go", "--type=renderer"}
	env := []string{"CEF_SUBPROCESS=1"}
	var got SubprocessRequest

	rt := New(Options{
		AppDir:          filepath.Join(t.TempDir(), "missing-app"),
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
			t.Fatal("bridge.Start called after subprocess hook handled request")
			return nil, nil
		}),
		SubprocessHook: func(_ context.Context, req SubprocessRequest) (SubprocessResult, error) {
			got = req
			return SubprocessResult{Handled: true, ExitCode: 9}, nil
		},
		Args:        args,
		Environment: env,
	})

	err := rt.Run(context.Background())
	var subprocessExit *SubprocessExit
	if !errors.As(err, &subprocessExit) {
		t.Fatalf("Run() error = %v, want SubprocessExit", err)
	}
	if subprocessExit.Code != 9 {
		t.Fatalf("SubprocessExit.Code = %d, want 9", subprocessExit.Code)
	}
	if rt.State() != StateStopped {
		t.Fatalf("State() = %s, want %s", rt.State(), StateStopped)
	}
	if !slices.Equal(got.Args, args) {
		t.Fatalf("hook Args = %#v, want %#v", got.Args, args)
	}
	if !slices.Equal(got.Environment, env) {
		t.Fatalf("hook Environment = %#v, want %#v", got.Environment, env)
	}
}

func TestNativeSubprocessHookHandlesCEFSubprocessArgs(t *testing.T) {
	result, err := NativeSubprocessHook(context.Background(), SubprocessRequest{
		Args: []string{"electron-go", "--type=renderer"},
	})
	if err != nil {
		t.Fatalf("NativeSubprocessHook() error = %v", err)
	}
	if !result.Handled {
		t.Fatal("NativeSubprocessHook() did not handle CEF subprocess args")
	}
	if result.ExitCode != native.CEFSubprocessUnavailableExitCode {
		t.Fatalf("ExitCode = %d, want %d", result.ExitCode, native.CEFSubprocessUnavailableExitCode)
	}
}

func TestNativeSubprocessHookIgnoresBrowserProcessArgs(t *testing.T) {
	result, err := NativeSubprocessHook(context.Background(), SubprocessRequest{
		Args: []string{"electron-go", "./app"},
	})
	if err != nil {
		t.Fatalf("NativeSubprocessHook() error = %v", err)
	}
	if result.Handled {
		t.Fatal("NativeSubprocessHook() handled browser process args")
	}
}

type bridgeFunc func(context.Context, native.StartRequest) (*native.StartResult, error)

func (f bridgeFunc) Start(ctx context.Context, req native.StartRequest) (*native.StartResult, error) {
	return f(ctx, req)
}

type mainPlanBridgeFake struct {
	browserID   int64
	startCalled bool
	initialized bool
	shutdown    bool
	create      native.BrowserWindowCreateRequest
	load        native.BrowserWindowLoadRequest
	close       native.BrowserWindowCloseRequest
	waited      bool
}

func (b *mainPlanBridgeFake) Start(context.Context, native.StartRequest) (*native.StartResult, error) {
	b.startCalled = true
	return &native.StartResult{}, nil
}

func (b *mainPlanBridgeFake) InitializeForStart(context.Context, native.StartRequest) error {
	time.Sleep(time.Millisecond)
	b.initialized = true
	return nil
}

func (b *mainPlanBridgeFake) Shutdown(context.Context) error {
	b.shutdown = true
	return nil
}

func (b *mainPlanBridgeFake) CreateBrowserWindow(_ context.Context, req native.BrowserWindowCreateRequest) (int64, error) {
	b.create = req
	return b.browserID, nil
}

func (b *mainPlanBridgeFake) LoadURL(_ context.Context, req native.BrowserWindowLoadRequest) error {
	b.load = req
	return nil
}

func (b *mainPlanBridgeFake) CloseBrowserWindow(_ context.Context, req native.BrowserWindowCloseRequest) error {
	b.close = req
	return nil
}

func (b *mainPlanBridgeFake) WaitForLoad(context.Context) error {
	b.waited = true
	return nil
}

func fixtureApp(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"fixture","version":"1.0.0","main":"main.js"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte(`console.log("fixture")`), 0o644); err != nil {
		t.Fatalf("write main.js: %v", err)
	}
	return dir
}

func scopedMainApp(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"name":"scoped","version":"1.0.0","main":"main.js"}`), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	source := `const { app, BrowserWindow } = require('electron')
const traceEnabled = process.env.ELECTRON_GO_BENCHMARK_TRACE === '1'
function mark(name) {}
function emitTrace() {}
async function main() {
  await app.whenReady()
  const win = new BrowserWindow({ width: 800, height: 600, show: true, webPreferences: { contextIsolation: true, sandbox: true } })
  win.webContents.once('did-finish-load', () => {
    mark('did_finish_load')
    setImmediate(() => {
      mark('quit_requested')
      emitTrace()
      win.close()
      app.quit()
    })
  })
  await win.loadFile('index.html')
}
app.on('window-all-closed', () => { app.quit() })
main()
`
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte(source), 0o644); err != nil {
		t.Fatalf("write main.js: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<html></html>`), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	return dir
}
