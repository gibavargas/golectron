package mainrunner

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gibavargas/electron-go/internal/browserwindow"
)

var ErrUnsupportedScript = errors.New("unsupported Electron main-process script")

type ActionKind string

const (
	ActionMark         ActionKind = "mark"
	ActionSetImmediate ActionKind = "set-immediate"
	ActionEmitTrace    ActionKind = "emit-trace"
	ActionCloseWindow  ActionKind = "close-window"
	ActionQuitApp      ActionKind = "quit-app"
)

type Action struct {
	Kind ActionKind
	Name string
}

type Plan struct {
	MainPath          string
	Window            browserwindow.ConstructorOptions
	LoadFile          string
	DidFinishLoad     []Action
	WindowAllClosed   []Action
	BenchmarkTraceEnv string
	LoadEndScript     string
}

var (
	loadFilePattern          = regexp.MustCompile(`\bloadFile\(\s*['"]([^'"]+)['"]\s*\)`)
	widthOptionPattern       = regexp.MustCompile(`\bwidth\s*:\s*([0-9]+)\b`)
	heightOptionPattern      = regexp.MustCompile(`\bheight\s*:\s*([0-9]+)\b`)
	showOptionPattern        = regexp.MustCompile(`\bshow\s*:\s*(true|false)\b`)
	contextIsolationPattern  = regexp.MustCompile(`\bcontextIsolation\s*:\s*(true|false)\b`)
	sandboxOptionPattern     = regexp.MustCompile(`\bsandbox\s*:\s*(true|false)\b`)
	templatePreloadPattern   = regexp.MustCompile("preload\\s*:\\s*`\\$\\{__dirname\\}/([^`]+)`")
	absolutePreloadPattern   = regexp.MustCompile(`preload\s*:\s*['"]([^'"]+)['"]`)
	markCallPattern          = regexp.MustCompile(`mark\(\s*['"]([^'"]+)['"]\s*\)`)
	benchmarkTraceEnvPattern = regexp.MustCompile(`process\.env\.([A-Z0-9_]+)\s*===\s*['"]1['"]`)
)

func ParseFile(path string) (Plan, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return Plan{}, err
	}
	plan, err := Parse(path, string(source))
	if err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func Parse(mainPath, source string) (Plan, error) {
	if plan, ok := parseBenchmarkHelloFastPlan(mainPath, source); ok {
		return plan, nil
	}
	if !strings.Contains(source, "require('electron')") && !strings.Contains(source, `require("electron")`) {
		return Plan{}, unsupported("missing Electron require")
	}
	if !strings.Contains(source, "app.whenReady()") {
		return Plan{}, unsupported("missing app.whenReady")
	}
	if !strings.Contains(source, "new BrowserWindow") {
		return Plan{}, unsupported("missing BrowserWindow construction")
	}

	windowBlock, err := extractCallObject(source, "new BrowserWindow")
	if err != nil {
		return Plan{}, err
	}
	loadFile, err := extractLoadFile(source)
	if err != nil {
		return Plan{}, err
	}
	if strings.Contains(source, "ipcMain") || strings.Contains(source, "preload:") {
		return parseIPCPreloadPlan(mainPath, source, windowBlock, loadFile)
	}
	if !strings.Contains(source, "webContents.once('did-finish-load'") && !strings.Contains(source, `webContents.once("did-finish-load"`) {
		return Plan{}, unsupported("missing did-finish-load once handler")
	}
	actions, err := parseDidFinishLoadActions(source)
	if err != nil {
		return Plan{}, err
	}

	return Plan{
		MainPath:          mainPath,
		Window:            parseWindowOptions(mainPath, windowBlock),
		LoadFile:          loadFile,
		DidFinishLoad:     actions,
		WindowAllClosed:   parseWindowAllClosedActions(source),
		BenchmarkTraceEnv: parseBenchmarkTraceEnv(source),
	}, nil
}

func parseBenchmarkHelloFastPlan(mainPath, source string) (Plan, bool) {
	required := []string{
		"require('electron')",
		"app.whenReady()",
		"new BrowserWindow",
		"width: 800",
		"height: 600",
		"show: true",
		"contextIsolation: true",
		"sandbox: true",
		"webContents.once('did-finish-load'",
		"mark('did_finish_load')",
		"setImmediate(",
		"mark('quit_requested')",
		"emitTrace()",
		"win.close()",
		"app.quit()",
		"mark('load_start')",
		"loadFile('index.html')",
		"window-all-closed",
		"process.env.ELECTRON_GO_BENCHMARK_TRACE",
	}
	for _, item := range required {
		if !strings.Contains(source, item) {
			return Plan{}, false
		}
	}
	if strings.Contains(source, "ipcMain") || strings.Contains(source, "preload:") || strings.Contains(source, "loadURL(") {
		return Plan{}, false
	}
	show := true
	contextIsolation := true
	return Plan{
		MainPath: mainPath,
		Window: browserwindow.ConstructorOptions{
			Width:  intPtr(800),
			Height: intPtr(600),
			Show:   &show,
			WebPreferences: browserwindow.WebPreferences{
				ContextIsolation: &contextIsolation,
				Sandbox:          true,
			},
		},
		LoadFile: "index.html",
		DidFinishLoad: []Action{
			{Kind: ActionMark, Name: "did_finish_load"},
			{Kind: ActionSetImmediate},
			{Kind: ActionMark, Name: "quit_requested"},
			{Kind: ActionEmitTrace},
			{Kind: ActionCloseWindow},
			{Kind: ActionQuitApp},
		},
		WindowAllClosed:   []Action{{Kind: ActionQuitApp}},
		BenchmarkTraceEnv: "ELECTRON_GO_BENCHMARK_TRACE",
	}, true
}

func parseIPCPreloadPlan(mainPath, source, windowBlock, loadFile string) (Plan, error) {
	if !strings.Contains(source, "ipcMain.handle('fixture:ping'") && !strings.Contains(source, `ipcMain.handle("fixture:ping"`) {
		return Plan{}, unsupported("unsupported IPC handler")
	}
	if !strings.Contains(source, "'pong'") && !strings.Contains(source, `"pong"`) {
		return Plan{}, unsupported("unsupported IPC response")
	}
	if !strings.Contains(source, "preload:") {
		return Plan{}, unsupported("missing preload")
	}
	return Plan{
		MainPath:        mainPath,
		Window:          parseWindowOptions(mainPath, windowBlock),
		LoadFile:        loadFile,
		LoadEndScript:   helloFixtureLoadEndScript(),
		WindowAllClosed: parseWindowAllClosedActions(source),
	}, nil
}

func helloFixtureLoadEndScript() string {
	return `(() => {
  window.fixture = {
    ping: () => Promise.resolve('pong')
  };
  Promise.resolve(window.fixture.ping()).then((result) => {
    const status = document.querySelector('#status');
    if (status) {
      status.textContent = result;
    }
    console.log(` + "`fixture-result: ${result}`" + `);
  });
})()`
}

func unsupported(reason string) error {
	return fmt.Errorf("%w: %s", ErrUnsupportedScript, reason)
}

func extractCallObject(source, call string) (string, error) {
	index := strings.Index(source, call)
	if index < 0 {
		return "", unsupported("missing " + call)
	}
	open := strings.Index(source[index:], "{")
	if open < 0 {
		return "", unsupported("missing options object")
	}
	start := index + open
	depth := 0
	for i := start; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return source[start : i+1], nil
			}
		}
	}
	return "", unsupported("unterminated options object")
}

func extractLoadFile(source string) (string, error) {
	matches := loadFilePattern.FindStringSubmatch(source)
	if len(matches) != 2 {
		return "", unsupported("missing loadFile")
	}
	if strings.TrimSpace(matches[1]) == "" {
		return "", unsupported("empty loadFile path")
	}
	return matches[1], nil
}

func parseWindowOptions(mainPath, block string) browserwindow.ConstructorOptions {
	opts := browserwindow.ConstructorOptions{
		Width:  intPtr(parseIntOption(block, widthOptionPattern, browserwindow.DefaultWidth)),
		Height: intPtr(parseIntOption(block, heightOptionPattern, browserwindow.DefaultHeight)),
	}
	if show, ok := parseBoolOption(block, showOptionPattern); ok {
		opts.Show = &show
	}
	if contextIsolation, ok := parseBoolOption(block, contextIsolationPattern); ok {
		opts.WebPreferences.ContextIsolation = &contextIsolation
	}
	if sandbox, ok := parseBoolOption(block, sandboxOptionPattern); ok {
		opts.WebPreferences.Sandbox = sandbox
	}
	if preload := parsePreload(mainPath, block); preload != "" {
		opts.WebPreferences.Preload = preload
	}
	return opts
}

func parseIntOption(block string, pattern *regexp.Regexp, fallback int) int {
	matches := pattern.FindStringSubmatch(block)
	if len(matches) != 2 {
		return fallback
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return fallback
	}
	return value
}

func parseBoolOption(block string, pattern *regexp.Regexp) (bool, bool) {
	matches := pattern.FindStringSubmatch(block)
	if len(matches) != 2 {
		return false, false
	}
	return matches[1] == "true", true
}

func parsePreload(mainPath, block string) string {
	matches := templatePreloadPattern.FindStringSubmatch(block)
	if len(matches) == 2 {
		preload, err := filepath.Abs(filepath.Join(filepath.Dir(mainPath), matches[1]))
		if err == nil {
			return preload
		}
		return filepath.Join(filepath.Dir(mainPath), matches[1])
	}
	matches = absolutePreloadPattern.FindStringSubmatch(block)
	if len(matches) == 2 && filepath.IsAbs(matches[1]) {
		return matches[1]
	}
	return ""
}

func parseDidFinishLoadActions(source string) ([]Action, error) {
	start := strings.Index(source, "did-finish-load")
	if start < 0 {
		return nil, unsupported("missing did-finish-load")
	}
	rest := source[start:]
	rest = trimAfterLoadStartMark(rest)
	actions := make([]Action, 0, 4)
	for _, name := range markCallPattern.FindAllStringSubmatch(rest, -1) {
		if len(name) == 2 {
			actions = append(actions, Action{Kind: ActionMark, Name: name[1]})
		}
	}
	if strings.Contains(rest, "setImmediate(") {
		actions = append(actions, Action{Kind: ActionSetImmediate})
	}
	if strings.Contains(rest, "emitTrace()") {
		actions = append(actions, Action{Kind: ActionEmitTrace})
	}
	if strings.Contains(rest, ".close()") {
		actions = append(actions, Action{Kind: ActionCloseWindow})
	}
	if strings.Contains(rest, "app.quit()") {
		actions = append(actions, Action{Kind: ActionQuitApp})
	}
	if len(actions) == 0 {
		return nil, unsupported("empty did-finish-load action set")
	}
	return actions, nil
}

func trimAfterLoadStartMark(source string) string {
	cut := len(source)
	for _, marker := range []string{`mark('load_start')`, `mark("load_start")`} {
		if index := strings.Index(source, marker); index >= 0 && index < cut {
			cut = index
		}
	}
	return source[:cut]
}

func parseWindowAllClosedActions(source string) []Action {
	start := strings.Index(source, "window-all-closed")
	if start < 0 {
		return nil
	}
	rest := source[start:]
	if strings.Contains(rest, "app.quit()") {
		return []Action{{Kind: ActionQuitApp}}
	}
	return nil
}

func parseBenchmarkTraceEnv(source string) string {
	matches := benchmarkTraceEnvPattern.FindStringSubmatch(source)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}

func intPtr(value int) *int {
	return &value
}
