package runtime

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/gibavargas/electron-go/internal/native"
)

func BenchmarkRuntimeStartStub(b *testing.B) {
	dir := fixtureApp(b)
	args := []string{"electron-go", dir}
	env := []string{"PATH=/test/bin"}
	for b.Loop() {
		rt := New(Options{
			AppDir:          dir,
			ElectronVersion: "42.0.0",
			Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
				return &native.StartResult{PID: 1, WindowCount: 1}, nil
			}),
			Args:        args,
			Environment: env,
		})
		if err := rt.Run(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRuntimeStartStubBorrowedArgsEnv(b *testing.B) {
	dir := fixtureApp(b)
	args := []string{"electron-go", dir}
	env := []string{"PATH=/test/bin"}
	for b.Loop() {
		rt := New(Options{
			AppDir:          dir,
			ElectronVersion: "42.0.0",
			Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
				return &native.StartResult{PID: 1, WindowCount: 1}, nil
			}),
			Args:          args,
			Environment:   env,
			BorrowArgsEnv: true,
		})
		if err := rt.Run(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRuntimeBenchmarkHelloStubBorrowedArgsEnv(b *testing.B) {
	dir := filepath.Join("..", "..", "compat", "fixtures", "benchmark-hello")
	args := []string{"electron-go", dir}
	env := []string{"PATH=/test/bin"}
	for b.Loop() {
		rt := New(Options{
			AppDir:          dir,
			ElectronVersion: "42.0.0",
			Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
				return &native.StartResult{PID: 1, WindowCount: 1}, nil
			}),
			Args:          args,
			Environment:   env,
			BorrowArgsEnv: true,
		})
		if err := rt.Run(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRuntimeBenchmarkHelloMainPlanBridge(b *testing.B) {
	dir := filepath.Join("..", "..", "compat", "fixtures", "benchmark-hello")
	args := []string{"electron-go", dir}
	env := []string{"PATH=/test/bin"}
	for b.Loop() {
		bridge := &benchmarkMainPlanBridge{browserID: 42}
		rt := New(Options{
			AppDir:            dir,
			ElectronVersion:   "42.0.0",
			Bridge:            bridge,
			Args:              args,
			Environment:       env,
			BorrowArgsEnv:     true,
			SkipFinalShutdown: true,
		})
		if err := rt.Run(context.Background()); err != nil {
			b.Fatal(err)
		}
		if !bridge.initialized || !bridge.waited || bridge.close.BrowserID != 42 {
			b.Fatalf("bridge = %#v, want initialized, waited, and closed browser 42", bridge)
		}
	}
}

type benchmarkMainPlanBridge struct {
	browserID   int64
	initialized bool
	waited      bool
	close       native.BrowserWindowCloseRequest
}

func (b *benchmarkMainPlanBridge) Start(context.Context, native.StartRequest) (*native.StartResult, error) {
	return &native.StartResult{}, nil
}

func (b *benchmarkMainPlanBridge) InitializeForStart(context.Context, native.StartRequest) error {
	b.initialized = true
	return nil
}

func (b *benchmarkMainPlanBridge) Shutdown(context.Context) error {
	return nil
}

func (b *benchmarkMainPlanBridge) CreateBrowserWindow(_ context.Context, _ native.BrowserWindowCreateRequest) (int64, error) {
	return b.browserID, nil
}

func (b *benchmarkMainPlanBridge) LoadURL(context.Context, native.BrowserWindowLoadRequest) error {
	return nil
}

func (b *benchmarkMainPlanBridge) CloseBrowserWindow(_ context.Context, req native.BrowserWindowCloseRequest) error {
	b.close = req
	return nil
}

func (b *benchmarkMainPlanBridge) WaitForLoad(context.Context) error {
	b.waited = true
	return nil
}
