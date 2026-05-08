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
