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
