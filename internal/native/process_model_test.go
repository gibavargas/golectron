package native

import (
	"strings"
	"testing"
)

func TestCEFProcessTypeFromArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "browser process", args: []string{"electron-go"}, want: ""},
		{name: "renderer equals", args: []string{"electron-go", "--type=renderer"}, want: "renderer"},
		{name: "utility split", args: []string{"electron-go", "--type", "utility"}, want: "utility"},
		{name: "crashpad executable", args: []string{"/tmp/chrome_crashpad_handler", "--database=/tmp/crashes"}, want: "crashpad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CEFProcessTypeFromArgs(tt.args); got != tt.want {
				t.Fatalf("CEFProcessTypeFromArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProcessTypesFromObservations(t *testing.T) {
	got := ProcessTypesFromObservations([]ObservedProcess{
		{PID: 10, Type: "renderer"},
		{PID: 11, Type: "utility"},
		{PID: 12, Type: "renderer"},
		{PID: 13},
	})
	want := []string{"renderer", "utility"}
	if len(got) != len(want) {
		t.Fatalf("ProcessTypesFromObservations() = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("ProcessTypesFromObservations() = %#v, want %#v", got, want)
		}
	}
}

func TestValidateRuntimeProcessModelReport(t *testing.T) {
	valid := RuntimeProcessModelReport{
		PID:                  100,
		MainThreadStable:     true,
		ContextInitialized:   true,
		BrowserID:            1,
		MainFrameLoaded:      true,
		WindowClosed:         true,
		ObservedProcessTypes: []string{"renderer"},
		NoZombieDescendants:  true,
	}

	tests := []struct {
		name   string
		mutate func(*RuntimeProcessModelReport)
		want   string
	}{
		{name: "valid"},
		{name: "pid", mutate: func(r *RuntimeProcessModelReport) { r.PID = 0 }, want: "PID"},
		{name: "thread", mutate: func(r *RuntimeProcessModelReport) { r.MainThreadStable = false }, want: "locked OS thread"},
		{name: "context", mutate: func(r *RuntimeProcessModelReport) { r.ContextInitialized = false }, want: "context"},
		{name: "browser", mutate: func(r *RuntimeProcessModelReport) { r.BrowserID = 0 }, want: "BrowserWindow"},
		{name: "load", mutate: func(r *RuntimeProcessModelReport) { r.MainFrameLoaded = false }, want: "main frame"},
		{name: "close", mutate: func(r *RuntimeProcessModelReport) { r.WindowClosed = false }, want: "close"},
		{name: "process types", mutate: func(r *RuntimeProcessModelReport) { r.ObservedProcessTypes = nil }, want: "child process types"},
		{name: "zombies", mutate: func(r *RuntimeProcessModelReport) { r.NoZombieDescendants = false }, want: "descendants"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := valid
			report.ObservedProcessTypes = append([]string(nil), valid.ObservedProcessTypes...)
			if tt.mutate != nil {
				tt.mutate(&report)
			}
			err := ValidateRuntimeProcessModelReport(report)
			if tt.want == "" {
				if err != nil {
					t.Fatalf("ValidateRuntimeProcessModelReport() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ValidateRuntimeProcessModelReport() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}
