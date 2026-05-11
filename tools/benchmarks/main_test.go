package main

import (
	"reflect"
	"testing"
)

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

func TestParseMaxRSSKB(t *testing.T) {
	darwin := "123456 maximum resident set size\n"
	if got := parseMaxRSSKB(darwin, "darwin"); got != 120 {
		t.Fatalf("parseMaxRSSKB(darwin) = %d, want 120", got)
	}

	linux := "Maximum resident set size (kbytes): 98765\n"
	if got := parseMaxRSSKB(linux, "linux"); got != 98765 {
		t.Fatalf("parseMaxRSSKB(linux) = %d, want 98765", got)
	}
}

func TestParseStartupTraceMS(t *testing.T) {
	output := "electron-go: loaded app 1.0.0\n" +
		"electron-go-startup-trace: {\"cef_initialize\":123,\"create_browser\":45}\n"
	got := parseStartupTraceMS(output)
	if got["cef_initialize"] != 123 || got["create_browser"] != 45 {
		t.Fatalf("parseStartupTraceMS() = %#v", got)
	}
}

func TestParseTraceMS(t *testing.T) {
	output := "noise\n" +
		"benchmark-trace: {\"app_ready\":12,\"did_finish_load\":89}\n"
	got := parseTraceMS(output, "benchmark-trace:")
	if got["app_ready"] != 12 || got["did_finish_load"] != 89 {
		t.Fatalf("parseTraceMS() = %#v", got)
	}
}

func TestSummarizeIncludesTraceMedians(t *testing.T) {
	summary := summarize([]Sample{
		{ExitCode: 0, DurationMS: 100, StartupTraceMS: map[string]int64{"cef_initialize": 10}, BenchmarkTraceMS: map[string]int64{"app_ready": 40}},
		{ExitCode: 0, DurationMS: 120, StartupTraceMS: map[string]int64{"cef_initialize": 14}, BenchmarkTraceMS: map[string]int64{"app_ready": 42}},
		{ExitCode: 0, DurationMS: 140, StartupTraceMS: map[string]int64{"cef_initialize": 12}, BenchmarkTraceMS: map[string]int64{"app_ready": 44}},
	}, false, false)
	if summary.StartupTraceMedianMS["cef_initialize"] != 12 {
		t.Fatalf("startup trace medians = %#v, want cef_initialize=12", summary.StartupTraceMedianMS)
	}
	if summary.BenchmarkTraceMedianMS["app_ready"] != 42 {
		t.Fatalf("benchmark trace medians = %#v, want app_ready=42", summary.BenchmarkTraceMedianMS)
	}
}

func TestComputeComparisons(t *testing.T) {
	results := []CommandResult{
		{
			Name:    "electron",
			Summary: Summary{DurationMedianMS: 200, MaxRSSMedianKB: 1000, ProcessTreeRSSPeakMedianKB: 5000},
		},
		{
			Name:    "electron-go",
			Summary: Summary{DurationMedianMS: 50, MaxRSSMedianKB: 250, ProcessTreeRSSPeakMedianKB: 2000},
		},
	}

	got := computeComparisons(results)
	if len(got) != 3 {
		t.Fatalf("len(comparisons) = %d, want 3", len(got))
	}
	if got[0].Metric != "duration_median_ms" || got[0].ElectronOverElectronGo != 4 {
		t.Fatalf("duration comparison = %#v, want 4x ElectronOverElectronGo", got[0])
	}
	if got[1].Metric != "max_rss_median_kb" || got[1].ElectronGoOverElectron != 0.25 {
		t.Fatalf("rss comparison = %#v, want 0.25 ElectronGoOverElectron", got[1])
	}
	if got[2].Metric != "process_tree_rss_peak_median_kb" || got[2].ElectronGoOverElectron != 0.4 {
		t.Fatalf("process tree rss comparison = %#v, want 0.4 ElectronGoOverElectron", got[2])
	}
}

func TestAlternatingScheduleBalancesFirstRunner(t *testing.T) {
	specs := []commandSpec{
		{Name: "electron", Command: "npx electron"},
		{Name: "electron-go", Command: "go run ./cmd/electron-go"},
	}
	argsByName := map[string][]string{
		"electron":    {"npx", "electron"},
		"electron-go": {"go", "run", "./cmd/electron-go"},
	}

	got := alternatingSchedule(specs, argsByName, 4)
	wantNames := []string{
		"electron", "electron-go",
		"electron-go", "electron",
		"electron", "electron-go",
		"electron-go", "electron",
	}
	if len(got) != len(wantNames) {
		t.Fatalf("len(schedule) = %d, want %d", len(got), len(wantNames))
	}
	for i, wantName := range wantNames {
		if got[i].Name != wantName {
			t.Fatalf("schedule[%d].Name = %q, want %q; schedule=%#v", i, got[i].Name, wantName, got)
		}
		wantSequence := i + 1
		if got[i].Sequence != wantSequence {
			t.Fatalf("schedule[%d].Sequence = %d, want %d", i, got[i].Sequence, wantSequence)
		}
		wantPair := i/2 + 1
		if got[i].Pair != wantPair || got[i].Iteration != wantPair {
			t.Fatalf("schedule[%d] pair/iteration = %d/%d, want %d/%d", i, got[i].Pair, got[i].Iteration, wantPair, wantPair)
		}
	}
}

func TestAlternatingScheduleSkipsUnavailableCommand(t *testing.T) {
	specs := []commandSpec{
		{Name: "electron", Command: ""},
		{Name: "electron-go", Command: "go run ./cmd/electron-go"},
	}
	argsByName := map[string][]string{
		"electron-go": {"go", "run", "./cmd/electron-go"},
	}

	got := alternatingSchedule(specs, argsByName, 2)
	if len(got) != 2 {
		t.Fatalf("len(schedule) = %d, want 2", len(got))
	}
	for i := range got {
		if got[i].Name != "electron-go" {
			t.Fatalf("schedule[%d].Name = %q, want electron-go", i, got[i].Name)
		}
		if got[i].Sequence != i+1 || got[i].Pair != i+1 || got[i].Iteration != i+1 {
			t.Fatalf("schedule[%d] = %#v, want matching sequence/pair/iteration", i, got[i])
		}
	}
}

func TestValidateRequiredFaster(t *testing.T) {
	report := Report{
		Iterations: 3,
		Results: []CommandResult{
			{Name: "electron", Summary: Summary{Successes: 3}},
			{Name: "electron-go", Summary: Summary{Successes: 3}},
		},
		Comparisons: []Comparison{
			{Metric: "duration_median_ms", Electron: 200, ElectronGo: 100, ElectronGoFasterOrLighter: true},
		},
	}

	if err := validateRequiredFaster(report, []string{"duration_median_ms"}); err != nil {
		t.Fatalf("validateRequiredFaster returned error: %v", err)
	}
}

func TestValidateRequiredFasterRejectsSlowElectronGo(t *testing.T) {
	report := Report{
		Iterations: 1,
		Results: []CommandResult{
			{Name: "electron", Summary: Summary{Successes: 1}},
			{Name: "electron-go", Summary: Summary{Successes: 1}},
		},
		Comparisons: []Comparison{
			{Metric: "duration_median_ms", Electron: 100, ElectronGo: 200, ElectronGoFasterOrLighter: false},
		},
	}

	if err := validateRequiredFaster(report, []string{"duration_median_ms"}); err == nil {
		t.Fatal("validateRequiredFaster returned nil error for slower Electron-Go")
	}
}

func TestValidateRequiredFasterRejectsFailures(t *testing.T) {
	report := Report{
		Iterations: 2,
		Results: []CommandResult{
			{Name: "electron", Summary: Summary{Successes: 2}},
			{Name: "electron-go", Summary: Summary{Successes: 1, Failures: 1}},
		},
		Comparisons: []Comparison{
			{Metric: "duration_median_ms", ElectronGoFasterOrLighter: true},
		},
	}

	if err := validateRequiredFaster(report, []string{"duration_median_ms"}); err == nil {
		t.Fatal("validateRequiredFaster returned nil error for failed samples")
	}
}
