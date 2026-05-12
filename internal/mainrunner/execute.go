package mainrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/gibavargas/electron-go/internal/applifecycle"
	"github.com/gibavargas/electron-go/internal/browserwindow"
	"github.com/gibavargas/electron-go/internal/native"
	"github.com/gibavargas/electron-go/internal/webcontents"
)

type Driver interface {
	CreateBrowserWindow(context.Context, native.BrowserWindowCreateRequest) (int64, error)
	LoadURL(context.Context, native.BrowserWindowLoadRequest) error
	CloseBrowserWindow(context.Context, native.BrowserWindowCloseRequest) error
}

type LoadEndScriptDriver interface {
	SetLoadEndScript(context.Context, native.BrowserWindowScriptRequest) error
}

type LoadWaitDriver interface {
	WaitForLoad(context.Context) error
}

type ExecuteOptions struct {
	Environment []string
	Out         io.Writer
	Trace       bool
	Events      bool
}

type Result struct {
	BrowserID int64
	TraceMS   map[string]int64
	Events    []string
	ExitCode  int
}

func Execute(ctx context.Context, plan Plan, driver Driver, opts ExecuteOptions) (Result, error) {
	if driver == nil {
		return Result{}, fmt.Errorf("main runner driver is required")
	}
	traceEnabled := opts.Trace || envEnabled(opts.Environment, plan.BenchmarkTraceEnv)
	var started time.Time
	var trace map[string]int64
	if traceEnabled {
		started = time.Now()
		trace = map[string]int64{}
	}
	mark := func(name string) {
		if trace != nil && name != "" {
			trace[name] = time.Since(started).Milliseconds()
		}
	}

	app := applifecycle.New()
	for _, action := range plan.WindowAllClosed {
		if action.Kind == ActionQuitApp {
			app.On(applifecycle.EventWindowAllClosed, func(*applifecycle.EventContext) {
				quitApp(app, 0)
			})
		}
	}
	if err := app.MarkReady(); err != nil {
		return Result{}, err
	}
	mark("app_ready")

	loadURL, err := loadFileURL(plan.MainPath, plan.LoadFile)
	if err != nil {
		return Result{}, err
	}

	normalized, err := browserwindow.NormalizeOptions(plan.Window)
	if err != nil {
		return Result{}, err
	}
	createURL := "about:blank"
	if plan.LoadEndScript == "" {
		createURL = loadURL
	}
	browserID, err := driver.CreateBrowserWindow(ctx, native.BrowserWindowCreateRequest{
		URL:             createURL,
		Width:           normalized.Width,
		Height:          normalized.Height,
		Show:            normalized.Show,
		AutoCloseOnLoad: false,
	})
	if err != nil {
		return Result{}, err
	}
	mark("window_created")

	window := browserwindow.NewWindow(browserID, normalized)
	var contents *webcontents.WebContents
	if opts.Events {
		contents = webcontents.New(browserID)
	}
	if plan.LoadEndScript != "" {
		scriptDriver, ok := driver.(LoadEndScriptDriver)
		if !ok {
			return Result{}, fmt.Errorf("main runner driver does not support load-end scripts")
		}
		if err := scriptDriver.SetLoadEndScript(ctx, native.BrowserWindowScriptRequest{BrowserID: browserID, Script: plan.LoadEndScript}); err != nil {
			return Result{}, err
		}
	}
	mark("load_start")
	if plan.LoadEndScript == "" {
		waitDriver, ok := driver.(LoadWaitDriver)
		if !ok {
			return Result{}, fmt.Errorf("main runner driver does not support load waits")
		}
		if err := waitDriver.WaitForLoad(ctx); err != nil {
			return Result{}, err
		}
	} else {
		if err := driver.LoadURL(ctx, native.BrowserWindowLoadRequest{BrowserID: browserID, URL: loadURL}); err != nil {
			return Result{}, err
		}
	}
	if contents != nil {
		if err := contents.LoadURL(loadURL); err != nil {
			return Result{}, err
		}
	}
	for _, action := range plan.DidFinishLoad {
		if err := executeAction(ctx, action, app, window, driver, trace, started); err != nil {
			return Result{}, err
		}
	}
	if app.IsQuitting() && opts.Out != nil && envEnabled(opts.Environment, plan.BenchmarkTraceEnv) {
		if err := emitBenchmarkTrace(opts.Out, trace); err != nil {
			return Result{}, err
		}
	}
	var events []string
	if opts.Events {
		events = collectEvents(app, window, contents)
	}
	return Result{
		BrowserID: browserID,
		TraceMS:   trace,
		Events:    events,
		ExitCode:  app.ExitCode(),
	}, nil
}

func executeAction(ctx context.Context, action Action, app *applifecycle.App, window *browserwindow.Window, driver Driver, trace map[string]int64, started time.Time) error {
	switch action.Kind {
	case ActionMark:
		if trace != nil && action.Name != "" {
			trace[action.Name] = time.Since(started).Milliseconds()
		}
	case ActionSetImmediate:
		return nil
	case ActionEmitTrace:
		return nil
	case ActionCloseWindow:
		if window.Close() {
			if err := driver.CloseBrowserWindow(ctx, native.BrowserWindowCloseRequest{BrowserID: window.ID()}); err != nil {
				return err
			}
			app.WindowAllClosed()
		}
	case ActionQuitApp:
		quitApp(app, 0)
	default:
		return fmt.Errorf("%w: action %q", ErrUnsupportedScript, action.Kind)
	}
	return nil
}

func quitApp(app *applifecycle.App, exitCode int) {
	if !app.IsQuitting() {
		app.Quit(exitCode)
	}
}

func loadFileURL(mainPath, loadFile string) (string, error) {
	if strings.TrimSpace(loadFile) == "" {
		return "", fmt.Errorf("%w: empty loadFile path", ErrUnsupportedScript)
	}
	path := loadFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(mainPath), path)
	}
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return "", err
		}
		path = abs
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String(), nil
}

func emitBenchmarkTrace(out io.Writer, trace map[string]int64) error {
	payload, err := json.Marshal(trace)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(out, "benchmark-trace: %s\n", payload)
	return err
}

func collectEvents(app *applifecycle.App, window *browserwindow.Window, contents *webcontents.WebContents) []string {
	var events []string
	for _, event := range app.Events() {
		events = append(events, "app:"+string(event))
	}
	for _, event := range window.Events() {
		events = append(events, "window:"+string(event))
	}
	if contents != nil {
		for _, event := range contents.Events() {
			events = append(events, "webContents:"+string(event))
		}
	}
	return events
}

func envEnabled(env []string, key string) bool {
	if key == "" {
		return false
	}
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			value := strings.TrimSpace(strings.TrimPrefix(entry, prefix))
			return value != "" && value != "0" && !strings.EqualFold(value, "false")
		}
	}
	return false
}
