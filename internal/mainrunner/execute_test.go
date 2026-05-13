package mainrunner

import (
	"bytes"
	"context"
	"io"
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
		Events:      true,
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.BrowserID != 42 || result.ExitCode != 0 {
		t.Fatalf("result = %#v, want browser 42 exit 0", result)
	}
	if !strings.HasSuffix(driver.create.URL, "/compat/fixtures/benchmark-hello/index.html") || driver.create.AutoCloseOnLoad {
		t.Fatalf("create request = %#v, want benchmark index.html without auto close", driver.create)
	}
	if driver.load.URL != "" {
		t.Fatalf("load URL = %q, want direct create without loadURL", driver.load.URL)
	}
	if !driver.waited {
		t.Fatal("driver did not wait for direct-created benchmark load")
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

func TestExecuteSkipsTraceAndEventsByDefault(t *testing.T) {
	plan, err := ParseFile("../../compat/fixtures/benchmark-hello/main.js")
	if err != nil {
		t.Fatalf("ParseFile() error = %v", err)
	}
	driver := &fakeDriver{browserID: 42}
	var out bytes.Buffer

	result, err := Execute(context.Background(), plan, driver, ExecuteOptions{Out: &out})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result.TraceMS != nil {
		t.Fatalf("TraceMS = %#v, want nil without trace env", result.TraceMS)
	}
	if result.Events != nil {
		t.Fatalf("Events = %#v, want nil without event collection", result.Events)
	}
	if !driver.create.AutoCloseOnLoad {
		t.Fatalf("create request = %#v, want benchmark fast path to auto-close on load", driver.create)
	}
	if out.String() != "" {
		t.Fatalf("stdout = %q, want no benchmark trace", out.String())
	}
}

func TestExecuteRequiresDriver(t *testing.T) {
	_, err := Execute(context.Background(), Plan{}, nil, ExecuteOptions{})
	if err == nil {
		t.Fatal("Execute() error = nil, want driver error")
	}
}

func BenchmarkExecuteBenchmarkPlanNoEvents(b *testing.B) {
	plan, err := ParseFile("../../compat/fixtures/benchmark-hello/main.js")
	if err != nil {
		b.Fatalf("ParseFile() error = %v", err)
	}
	ctx := context.Background()
	var driver fakeDriver
	b.ReportAllocs()
	for b.Loop() {
		driver = fakeDriver{browserID: 42}
		result, err := Execute(ctx, plan, &driver, ExecuteOptions{})
		if err != nil {
			b.Fatal(err)
		}
		if result.BrowserID != 42 || result.ExitCode != 0 || !driver.waited {
			b.Fatalf("result = %#v waited=%v, want browser 42 exit 0 waited", result, driver.waited)
		}
	}
}

func BenchmarkExecuteBenchmarkPlanWithTrace(b *testing.B) {
	plan, err := ParseFile("../../compat/fixtures/benchmark-hello/main.js")
	if err != nil {
		b.Fatalf("ParseFile() error = %v", err)
	}
	ctx := context.Background()
	opts := ExecuteOptions{
		Environment: []string{"ELECTRON_GO_BENCHMARK_TRACE=1"},
		Out:         io.Discard,
	}
	var driver fakeDriver
	b.ReportAllocs()
	for b.Loop() {
		driver = fakeDriver{browserID: 42}
		result, err := Execute(ctx, plan, &driver, opts)
		if err != nil {
			b.Fatal(err)
		}
		if result.BrowserID != 42 || result.ExitCode != 0 || result.TraceMS == nil || !driver.waited {
			b.Fatalf("result = %#v waited=%v, want traced browser 42 exit 0 waited", result, driver.waited)
		}
	}
}

type fakeDriver struct {
	browserID int64
	create    native.BrowserWindowCreateRequest
	load      native.BrowserWindowLoadRequest
	close     native.BrowserWindowCloseRequest
	script    native.BrowserWindowScriptRequest
	waited    bool
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

func (d *fakeDriver) WaitForLoad(context.Context) error {
	d.waited = true
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
