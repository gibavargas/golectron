package runtime

import (
	"context"
	"testing"

	"github.com/gibavargas/electron-go/internal/native"
)

func BenchmarkRuntimeStartStub(b *testing.B) {
	dir := fixtureApp(b)
	for b.Loop() {
		rt := New(Options{
			AppDir:          dir,
			ElectronVersion: "42.0.0",
			Bridge: bridgeFunc(func(context.Context, native.StartRequest) (*native.StartResult, error) {
				return &native.StartResult{PID: 1, WindowCount: 1}, nil
			}),
		})
		if err := rt.Run(context.Background()); err != nil {
			b.Fatal(err)
		}
	}
}
