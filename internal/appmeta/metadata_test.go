package appmeta

import (
	"os"
	"path/filepath"
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

func writePackage(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
}
