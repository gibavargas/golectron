package diagnostics

import (
	"errors"
	"reflect"
	"testing"
)

func TestCrashReporterCapturesReportsWithDynamicKeys(t *testing.T) {
	reporter, err := StartCrashReporter(CrashReporterOptions{
		SubmitURL:       "https://crash.example.test",
		UploadToServer:  true,
		ExtraParameters: map[string]string{"channel": "stable"},
	})
	if err != nil {
		t.Fatalf("StartCrashReporter() error = %v", err)
	}
	if err := reporter.AddExtraParameter("build", "42"); err != nil {
		t.Fatalf("AddExtraParameter() error = %v", err)
	}
	if err := reporter.Capture(CrashReport{ProcessType: "renderer", Reason: "oom", Stack: []string{"main.js:1"}}); err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	reports := reporter.Reports()
	if len(reports) != 1 {
		t.Fatalf("Reports() len = %d, want 1", len(reports))
	}
	if reports[0].ExtraParameters["channel"] != "stable" || reports[0].ExtraParameters["build"] != "42" {
		t.Fatalf("parameters = %#v", reports[0].ExtraParameters)
	}
	reports[0].Stack[0] = "mutated"
	if got := reporter.Reports()[0].Stack[0]; got != "main.js:1" {
		t.Fatalf("Reports() returned mutable stack: %q", got)
	}
}

func TestCrashReporterValidation(t *testing.T) {
	if _, err := StartCrashReporter(CrashReporterOptions{UploadToServer: true}); !errors.Is(err, ErrInvalidCrashReporter) {
		t.Fatalf("StartCrashReporter(missing url) error = %v, want ErrInvalidCrashReporter", err)
	}
	reporter, err := StartCrashReporter(CrashReporterOptions{})
	if err != nil {
		t.Fatalf("StartCrashReporter() error = %v", err)
	}
	if err := reporter.AddExtraParameter("bad\nkey", "x"); !errors.Is(err, ErrInvalidCrashReporter) {
		t.Fatalf("AddExtraParameter(invalid) error = %v, want ErrInvalidCrashReporter", err)
	}
	if err := reporter.Capture(CrashReport{ProcessType: "renderer"}); !errors.Is(err, ErrInvalidCrashReporter) {
		t.Fatalf("Capture(invalid) error = %v, want ErrInvalidCrashReporter", err)
	}
}

func TestTracerLifecycleAndHeapProfile(t *testing.T) {
	var tracer Tracer
	if _, err := tracer.Stop(""); !errors.Is(err, ErrTracingInactive) {
		t.Fatalf("Stop(inactive) error = %v, want ErrTracingInactive", err)
	}
	if err := tracer.Start(TraceConfig{Categories: []string{"v8", "blink"}, HeapProfiling: true}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := tracer.Start(TraceConfig{}); !errors.Is(err, ErrTracingActive) {
		t.Fatalf("Start(active) error = %v, want ErrTracingActive", err)
	}
	result, err := tracer.Stop("trace.json")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	want := TraceResult{Categories: []string{"v8", "blink"}, HeapProfile: true, Path: "trace.json"}
	if !reflect.DeepEqual(result, want) {
		t.Fatalf("Stop() = %#v, want %#v", result, want)
	}
	result.Categories[0] = "mutated"
	if got := tracer.Results()[0].Categories[0]; got != "v8" {
		t.Fatalf("Results() returned mutable categories: %q", got)
	}
}

func TestTracerDefaultsCategoriesAndPath(t *testing.T) {
	var tracer Tracer
	if err := tracer.Start(TraceConfig{}); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	result, err := tracer.Stop("")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if result.Path != "trace.json" || len(result.Categories) != 1 || result.Categories[0] != "*" {
		t.Fatalf("default result = %#v", result)
	}
}
