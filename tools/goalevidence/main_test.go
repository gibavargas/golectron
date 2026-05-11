package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildEvidenceExtractsBenchmarkRatio(t *testing.T) {
	reportPath := writeBenchmarkReport(t, `{
  "comparisons": [
    {
      "metric": "duration_median_ms",
      "electron_go_over_electron": 0.7912
    }
  ]
}`)

	got, err := buildEvidence(reportPath, "https://example.test/run/1", "abc123", "duration_median_ms", 0.5)
	if err != nil {
		t.Fatalf("buildEvidence() error = %v", err)
	}
	if got.Performance.TargetElectronGoOverElectron != 0.5 {
		t.Fatalf("target = %v, want 0.5", got.Performance.TargetElectronGoOverElectron)
	}
	if got.Performance.LatestElectronGoOverElectron != 0.7912 {
		t.Fatalf("latest = %v, want 0.7912", got.Performance.LatestElectronGoOverElectron)
	}
	if got.Performance.LatestRunURL != "https://example.test/run/1" || got.Performance.LatestCommit != "abc123" {
		t.Fatalf("evidence metadata = %#v", got.Performance)
	}
}

func TestBuildEvidenceRejectsMissingMetric(t *testing.T) {
	reportPath := writeBenchmarkReport(t, `{"comparisons":[]}`)
	if _, err := buildEvidence(reportPath, "https://example.test/run/1", "abc123", "duration_median_ms", 0.5); err == nil {
		t.Fatal("buildEvidence() error = nil, want missing metric error")
	}
}

func TestRunWritesGoalEvidenceJSON(t *testing.T) {
	reportPath := writeBenchmarkReport(t, `{
  "comparisons": [
    {
      "metric": "duration_median_ms",
      "electron_go_over_electron": 0.42
    }
  ]
}`)
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--benchmark", reportPath,
		"--run-url", "https://example.test/run/2",
		"--commit", "def456",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() exit = %d stderr=%s", code, stderr.String())
	}
	var payload goalEvidence
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("output is not goal evidence JSON: %v\n%s", err, stdout.String())
	}
	if payload.Performance.LatestElectronGoOverElectron != 0.42 {
		t.Fatalf("latest ratio = %v, want 0.42", payload.Performance.LatestElectronGoOverElectron)
	}
}

func TestRunRejectsMissingRequiredInputs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run(no args) exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--benchmark is required") {
		t.Fatalf("stderr = %q, want benchmark error", stderr.String())
	}
}

func writeBenchmarkReport(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "benchmark.json")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write benchmark report: %v", err)
	}
	return path
}
