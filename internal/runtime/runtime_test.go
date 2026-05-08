package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	var got native.StartRequest
	rt := New(Options{
		AppDir:          dir,
		ElectronVersion: "42.0.0",
		Bridge: bridgeFunc(func(_ context.Context, req native.StartRequest) (*native.StartResult, error) {
			got = req
			return &native.StartResult{PID: 100, WindowCount: 1, Chromium: "148.0.7778.96", Node: "24.15.0", V8: "14.8.178.14"}, nil
		}),
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
