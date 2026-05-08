package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunResolvesFromIndexFile(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "CEF_VERSION")
	index := filepath.Join(dir, "index.json")
	if err := os.WriteFile(manifest, []byte(`
PLATFORM=linux64
DISTRIBUTION=minimal
CHROMIUM_MAJOR=148
CHROMIUM_VERSION=148.0.7778.96
`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(index, []byte(`{
  "linux64": {
    "versions": [
      {
        "cef_version": "148.0.1+gabc1234+chromium-148.0.7778.96",
        "channel": "stable",
        "chromium_version": "148.0.7778.96",
        "files": [
          {"name": "cef_binary_148.0.1+gabc1234+chromium-148.0.7778.96_linux64_minimal.tar.bz2", "sha1": "abc", "size": 42, "type": "minimal"}
        ]
      }
    ]
  }
}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	report, code := run(manifest, "", index, time.Second, false)
	if code != 0 {
		t.Fatalf("code = %d, report = %#v", code, report)
	}
	if !report.Resolved || report.Artifact == nil {
		t.Fatalf("report not resolved: %#v", report)
	}
}

func TestRunReportsNoMatch(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "CEF_VERSION")
	index := filepath.Join(dir, "index.json")
	if err := os.WriteFile(manifest, []byte(`
PLATFORM=linux64
DISTRIBUTION=minimal
CHROMIUM_MAJOR=148
CHROMIUM_VERSION=148.0.7778.96
`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(index, []byte(`{"linux64":{"versions":[]}}`), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	report, code := run(manifest, "", index, time.Second, false)
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if report.Error == "" {
		t.Fatalf("report.Error is empty")
	}
}
