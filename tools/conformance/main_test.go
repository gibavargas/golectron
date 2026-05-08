package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseOptions(t *testing.T) {
	opts, err := parseOptions([]string{
		"--fixture", "compat/fixtures/hello, compat/fixtures/ipc",
		"--fixture", "compat/fixtures/hello",
		"--electron", `npx electron "./shim app"`,
		"--electron-go", "go run ./cmd/electron-go",
		"--timeout", "5s",
		"--output", "report.json",
		"--allow-mismatch",
	})
	if err != nil {
		t.Fatalf("parseOptions returned error: %v", err)
	}

	wantFixtures := []string{"compat/fixtures/hello", "compat/fixtures/ipc"}
	if !reflect.DeepEqual(opts.Fixtures, wantFixtures) {
		t.Fatalf("fixtures = %#v, want %#v", opts.Fixtures, wantFixtures)
	}
	if opts.Electron != `npx electron "./shim app"` {
		t.Fatalf("electron = %q", opts.Electron)
	}
	if opts.ElectronGo != "go run ./cmd/electron-go" {
		t.Fatalf("electron-go = %q", opts.ElectronGo)
	}
	if opts.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", opts.Timeout)
	}
	if opts.Output != "report.json" {
		t.Fatalf("output = %q, want report.json", opts.Output)
	}
	if !opts.AllowMismatch {
		t.Fatal("allow mismatch = false, want true")
	}
}

func TestParseOptionsRequiresFixture(t *testing.T) {
	if _, err := parseOptions(nil); err == nil {
		t.Fatal("parseOptions returned nil error without fixture")
	}
}

func TestSplitCommandHandlesQuotesAndEscapes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "simple",
			in:   "go run ./cmd/electron-go",
			want: []string{"go", "run", "./cmd/electron-go"},
		},
		{
			name: "quoted argument",
			in:   `npx electron "./fixtures/hello app" --flag='two words'`,
			want: []string{"npx", "electron", "./fixtures/hello app", "--flag=two words"},
		},
		{
			name: "escaped space",
			in:   `node ./my\ app/main.js`,
			want: []string{"node", "./my app/main.js"},
		},
		{
			name: "empty quoted argument",
			in:   `cmd "" '' tail`,
			want: []string{"cmd", "", "", "tail"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitCommand(tt.in)
			if err != nil {
				t.Fatalf("splitCommand returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSplitCommandRejectsUnclosedQuotes(t *testing.T) {
	if _, err := splitCommand(`npx electron "fixture`); err == nil {
		t.Fatal("splitCommand returned nil error for unterminated quote")
	}
}

func TestParseFixtures(t *testing.T) {
	got := parseFixtures(" compat/fixtures/hello,compat/fixtures/ipc,, compat/fixtures/window ")
	want := []string{"compat/fixtures/hello", "compat/fixtures/ipc", "compat/fixtures/window"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFixtures() = %#v, want %#v", got, want)
	}
}

func TestCompareResultsMatch(t *testing.T) {
	electron := CommandResult{
		ExitCode: 0,
		Stdout:   "ready\n",
		Stderr:   "",
		TimedOut: false,
	}
	electronGo := CommandResult{
		ExitCode: 0,
		Stdout:   "ready\n",
		Stderr:   "",
		TimedOut: false,
	}

	got := compareResults(electron, electronGo)
	if !got.Match {
		t.Fatalf("Match = false, differences = %#v", got.Differences)
	}
	if len(got.Differences) != 0 {
		t.Fatalf("differences = %#v, want empty", got.Differences)
	}
}

func TestCompareResultsReportsDifferences(t *testing.T) {
	electron := CommandResult{
		ExitCode: 0,
		Stdout:   "ready\n",
		Stderr:   "",
		TimedOut: false,
	}
	electronGo := CommandResult{
		ExitCode: 1,
		Stdout:   "",
		Stderr:   "native bridge unavailable\n",
		TimedOut: true,
		Error:    "exit status 1",
	}

	got := compareResults(electron, electronGo)
	want := []string{"exit_code", "stdout", "stderr", "timed_out", "error"}
	if got.Match {
		t.Fatal("Match = true, want false")
	}
	if !reflect.DeepEqual(got.Differences, want) {
		t.Fatalf("differences = %#v, want %#v", got.Differences, want)
	}
}
