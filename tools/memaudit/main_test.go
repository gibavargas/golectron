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
