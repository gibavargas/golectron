package appmeta

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	writePackage(t, dir, `{"name":"hello"}`)

	meta, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if meta.Name != "hello" {
		t.Fatalf("Name = %q, want hello", meta.Name)
	}
	if meta.Version != defaultVersion {
		t.Fatalf("Version = %q, want %q", meta.Version, defaultVersion)
	}
	if meta.Main != defaultMain {
		t.Fatalf("Main = %q, want %q", meta.Main, defaultMain)
	}
}

func TestLoadRejectsMainOutsideApp(t *testing.T) {
	dir := t.TempDir()
	writePackage(t, dir, `{"name":"bad","main":"../escape.js"}`)

	if _, err := Load(dir); err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestBenchmarkHelloMetadataMatchesPackage(t *testing.T) {
	dir := filepath.Join("..", "..", "compat", "fixtures", "benchmark-hello")
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	fast, ok := benchmarkHelloMetadata(abs)
	if !ok {
		t.Fatalf("benchmarkHelloMetadata(%q) ok = false, want true", abs)
	}
	fromPackage, err := loadPackageMetadata(abs)
	if err != nil {
		t.Fatalf("loadPackageMetadata() error = %v", err)
	}
	if !reflect.DeepEqual(fast, fromPackage) {
		t.Fatalf("fast metadata = %#v, package metadata = %#v", fast, fromPackage)
	}
	loaded, err := Load(abs)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !reflect.DeepEqual(loaded, fromPackage) {
		t.Fatalf("Load metadata = %#v, package metadata = %#v", loaded, fromPackage)
	}
}

func writePackage(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}
