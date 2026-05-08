package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanRootFindsMemoryHotspots(t *testing.T) {
	dir := t.TempDir()
	body := `package fixture

import (
	"sync"
	"unsafe"
)

type payload struct {
	value any
	raw interface{}
}

var pool sync.Pool
var _ = unsafe.Sizeof(payload{})
`
	if err := os.WriteFile(filepath.Join(dir, "fixture.go"), []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	report, err := scanRoot(dir)
	if err != nil {
		t.Fatalf("scanRoot() error = %v", err)
	}
	if report.FilesScanned != 1 {
		t.Fatalf("FilesScanned = %d, want 1", report.FilesScanned)
	}
	if report.AnyUses != 1 {
		t.Fatalf("AnyUses = %d, want 1", report.AnyUses)
	}
	if report.EmptyInterfaceUses != 1 {
		t.Fatalf("EmptyInterfaceUses = %d, want 1", report.EmptyInterfaceUses)
	}
	if report.UnsafeImports != 1 || !report.UnsafeImportDetected {
		t.Fatalf("unsafe import not detected: %#v", report)
	}
	if report.SyncPoolUses != 1 {
		t.Fatalf("SyncPoolUses = %d, want 1", report.SyncPoolUses)
	}
}

func TestShouldSkipDir(t *testing.T) {
	if !shouldSkipDir(".git") {
		t.Fatal(".git should be skipped")
	}
	if shouldSkipDir("internal") {
		t.Fatal("internal should not be skipped")
	}
}

func TestCheckCEFLayoutReportsMissingFiles(t *testing.T) {
	layout := checkCEFLayout(t.TempDir())
	if layout.Valid {
		t.Fatal("layout.Valid = true, want false")
	}
	if len(layout.Missing) == 0 {
		t.Fatal("layout.Missing is empty")
	}
}

func TestCheckCEFLayoutValid(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{
		"libcef.so",
		"icudtl.dat",
		"v8_context_snapshot.bin",
		"chrome_100_percent.pak",
		"resources.pak",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	locales := filepath.Join(dir, "locales")
	if err := os.Mkdir(locales, 0o755); err != nil {
		t.Fatalf("mkdir locales: %v", err)
	}
	if err := os.WriteFile(filepath.Join(locales, "en-US.pak"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write locale: %v", err)
	}

	layout := checkCEFLayout(dir)
	if !layout.Valid {
		t.Fatalf("layout.Valid = false, missing %#v", layout.Missing)
	}
}
