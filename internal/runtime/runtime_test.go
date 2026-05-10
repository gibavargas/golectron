package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

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
