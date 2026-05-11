package mainrunner

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/gibavargas/electron-go/internal/native"
)

func TestExecuteRunsBenchmarkPlanThroughDriver(t *testing.T) {
	plan, err := ParseFile("../../compat/fixtures/benchmark-hello/main.js")
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	driver := &fakeDriver{browserID: 42}
	var out bytes.Buffer

	result, err := Execute(context.Background(), plan, driver, ExecuteOptions{
		Environment: []string{"ELECTRON_GO_BENCHMARK_TRACE=1"},
		Out:         &out,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.BrowserID != 42 || result.ExitCode != 0 {
		t.Fatalf("result = %#v, want browser 42 exit 0", result)
	}
	if driver.create.URL != "about:blank" || driver.create.AutoCloseOnLoad {
		t.Fatalf("create request = %#v, want about:blank without auto close", driver.create)
	}
	if !strings.HasSuffix(driver.load.URL, "/compat/fixtures/benchmark-hello/index.html") {
		t.Fatalf("load URL = %q, want benchmark index.html", driver.load.URL)
	}
	if driver.close.BrowserID != 42 {
		t.Fatalf("close request = %#v, want browser 42", driver.close)
	}
	assertEvent(t, result.Events, "app:ready")
	assertEvent(t, result.Events, "webContents:did-finish-load")
	assertEvent(t, result.Events, "window:closed")
	if !strings.Contains(out.String(), "benchmark-trace:") {
		t.Fatalf("stdout = %q, want benchmark trace", out.String())
	}
	for _, name := range []string{"app_ready", "window_created", "load_start", "did_finish_load", "quit_requested"} {
		if _, ok := result.TraceMS[name]; !ok {
			t.Fatalf("TraceMS = %#v, missing %s", result.TraceMS, name)
		}
	}
}

func TestExecuteInstallsHelloLoadEndScript(t *testing.T) {
	plan, err := ParseFile("../../compat/fixtures/hello/main.js")
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	driver := &fakeDriver{browserID: 42}

	result, err := Execute(context.Background(), plan, driver, ExecuteOptions{})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.BrowserID != 42 || result.ExitCode != 0 {
		t.Fatalf("result = %#v, want browser 42 exit 0", result)
	}
	if !strings.Contains(driver.script.Script, "fixture-result") {
		t.Fatalf("script request = %#v, want fixture-result script", driver.script)
	}
	if driver.script.BrowserID != 42 {
		t.Fatalf("script BrowserID = %d, want 42", driver.script.BrowserID)
	}
}

func TestExecuteRequiresDriver(t *testing.T) {
	_, err := Execute(context.Background(), Plan{}, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("Execute() error = nil, want driver error")
	}
}

type fakeDriver struct {
	browserID int64
	create    native.BrowserWindowCreateRequest
	load      native.BrowserWindowLoadRequest
	close     native.BrowserWindowCloseRequest
	script    native.BrowserWindowScriptRequest
}

func (d *fakeDriver) CreateBrowserWindow(_ context.Context, req native.BrowserWindowCreateRequest) (int64, error) {
	d.create = req
	return d.browserID, nil
}

func (d *fakeDriver) LoadURL(_ context.Context, req native.BrowserWindowLoadRequest) error {
	d.load = req
	return nil
}

func (d *fakeDriver) CloseBrowserWindow(_ context.Context, req native.BrowserWindowCloseRequest) error {
	d.close = req
	return nil
}

func (d *fakeDriver) SetLoadEndScript(_ context.Context, req native.BrowserWindowScriptRequest) error {
	d.script = req
	return nil
}

func assertEvent(t *testing.T, events []string, want string) {
	t.Helper()
	for _, event := range events {
		if event == want {
			return
		}
	}
	t.Fatalf("events = %#v, missing %q", events, want)
}
