package mainrunner

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/gibavargas/electron-go/internal/browserwindow"
)

func TestParseBenchmarkHelloPlan(t *testing.T) {
	mainPath := filepath.Join("..", "..", "compat", "fixtures", "benchmark-hello", "main.js")
	plan, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if plan.MainPath != mainPath {
		t.Fatalf("MainPath = %q, want %q", plan.MainPath, mainPath)
	}
	if plan.LoadFile != "index.html" {
		t.Fatalf("LoadFile = %q, want index.html", plan.LoadFile)
	}
	if plan.BenchmarkTraceEnv != "ELECTRON_GO_BENCHMARK_TRACE" {
		t.Fatalf("BenchmarkTraceEnv = %q, want ELECTRON_GO_BENCHMARK_TRACE", plan.BenchmarkTraceEnv)
	}
	normalized, err := browserwindow.NormalizeOptions(plan.Window)
	if err != nil {
		t.Fatalf("NormalizeOptions() error = %v", err)
	}
	if normalized.Width != 800 || normalized.Height != 600 || !normalized.Show {
		t.Fatalf("window = %#v, want visible 800x600", normalized)
	}
	if !normalized.WebPreferences.ContextIsolation || !normalized.WebPreferences.Sandbox {
		t.Fatalf("webPreferences = %#v, want contextIsolation and sandbox", normalized.WebPreferences)
	}
	assertAction(t, plan.DidFinishLoad, ActionMark, "did_finish_load")
	assertAction(t, plan.DidFinishLoad, ActionSetImmediate, "")
	assertAction(t, plan.DidFinishLoad, ActionMark, "quit_requested")
	assertAction(t, plan.DidFinishLoad, ActionEmitTrace, "")
	assertAction(t, plan.DidFinishLoad, ActionCloseWindow, "")
	assertAction(t, plan.DidFinishLoad, ActionQuitApp, "")
	assertAction(t, plan.WindowAllClosed, ActionQuitApp, "")
}

func TestParseRejectsIPCPreloadFixture(t *testing.T) {
	mainPath := filepath.Join("..", "..", "compat", "fixtures", "hello", "main.js")
	_, err := ParseFile(mainPath)
	if !errors.Is(err, ErrUnsupportedScript) {
		t.Fatalf("ParseFile() error = %v, want ErrUnsupportedScript", err)
	}
}

func TestParseRejectsMissingDidFinishLoad(t *testing.T) {
	source := `
const { app, BrowserWindow } = require('electron')
async function main() {
  await app.whenReady()
  const win = new BrowserWindow({ width: 800, height: 600 })
  await win.loadFile('index.html')
}
main()
`
	_, err := Parse("main.js", source)
	if !errors.Is(err, ErrUnsupportedScript) {
		t.Fatalf("Parse() error = %v, want ErrUnsupportedScript", err)
	}
}

func assertAction(t *testing.T, actions []Action, kind ActionKind, name string) {
	t.Helper()
	for _, action := range actions {
		if action.Kind == kind && action.Name == name {
			return
		}
	}
	t.Fatalf("actions = %#v, missing %s/%q", actions, kind, name)
}
