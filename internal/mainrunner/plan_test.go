package mainrunner

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
	if !strings.HasSuffix(plan.LoadURL, "/compat/fixtures/benchmark-hello/index.html") {
		t.Fatalf("LoadURL = %q, want benchmark index.html URL", plan.LoadURL)
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
	if plan.NormalizedWindow == nil {
		t.Fatal("NormalizedWindow is nil, want benchmark fast-path normalized window")
	}
	if *plan.NormalizedWindow != normalized {
		t.Fatalf("NormalizedWindow = %#v, want %#v", *plan.NormalizedWindow, normalized)
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
	assertNoAction(t, plan.DidFinishLoad, ActionMark, "load_start")
	assertAction(t, plan.WindowAllClosed, ActionQuitApp, "")
}

func TestParseHelloIPCPreloadPlan(t *testing.T) {
	mainPath := filepath.Join("..", "..", "compat", "fixtures", "hello", "main.js")
	plan, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if plan.LoadFile != "index.html" {
		t.Fatalf("LoadFile = %q, want index.html", plan.LoadFile)
	}
	if plan.Window.WebPreferences.Preload == "" {
		t.Fatalf("preload path is empty")
	}
	if plan.LoadEndScript == "" || !strings.Contains(plan.LoadEndScript, "fixture-result") {
		t.Fatalf("LoadEndScript = %q, want fixture-result injection", plan.LoadEndScript)
	}
}

func TestParseBenchmarkHelloPathPlanMatchesSourcePlan(t *testing.T) {
	mainPath := filepath.Join("..", "..", "compat", "fixtures", "benchmark-hello", "main.js")
	source, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	fromSource, err := Parse(mainPath, string(source))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	fromPath, err := ParseFile(mainPath)
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	if !reflect.DeepEqual(fromPath, fromSource) {
		t.Fatalf("path plan = %#v, source plan = %#v", fromPath, fromSource)
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

func assertNoAction(t *testing.T, actions []Action, kind ActionKind, name string) {
	t.Helper()
	for _, action := range actions {
		if action.Kind == kind && action.Name == name {
			t.Fatalf("actions = %#v, unexpectedly found %s/%q", actions, kind, name)
		}
	}
}
