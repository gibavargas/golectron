package asar

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestSplitAsarPath(t *testing.T) {
	path := filepath.Join("app", "resources", "app.asar", "lib", "main.js")
	got, ok := Split(path)
	if !ok {
		t.Fatal("Split() ok = false, want true")
	}
	if got.ArchivePath != filepath.Join("app", "resources", "app.asar") || got.InnerPath != "lib/main.js" {
		t.Fatalf("Split() = %#v", got)
	}
	if got.UnpackedPath != filepath.Join("app", "resources", "app.asar.unpacked", "lib", "main.js") {
		t.Fatalf("UnpackedPath = %q", got.UnpackedPath)
	}
}

func TestArchiveStatAndCopySource(t *testing.T) {
	archive, err := NewArchive(filepath.Join("resources", "app.asar"), []Entry{
		{Path: "lib/main.js", Size: 10, Mode: 0o644},
		{Path: "native/addon.node", Size: 20, Unpacked: true},
		{Path: "assets", Dir: true},
	})
	if err != nil {
		t.Fatalf("NewArchive() error = %v", err)
	}
	entry, err := archive.Stat("/lib/main.js")
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if entry.Size != 10 || entry.Mode != 0o644 {
		t.Fatalf("entry = %#v", entry)
	}
	source, err := archive.CopyFileSource("native/addon.node")
	if err != nil {
		t.Fatalf("CopyFileSource(unpacked) error = %v", err)
	}
	if source != filepath.Join("resources", "app.asar.unpacked", "native", "addon.node") {
		t.Fatalf("unpacked source = %q", source)
	}
	if _, err := archive.CopyFileSource("assets"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("CopyFileSource(dir) error = %v, want ErrInvalidPath", err)
	}
}

func TestResolveModule(t *testing.T) {
	archive, err := NewArchive("app.asar", []Entry{
		{Path: "main.js"},
		{Path: "pkg/index.js"},
		{Path: "native/addon.node", Unpacked: true},
	})
	if err != nil {
		t.Fatalf("NewArchive() error = %v", err)
	}
	if got, err := archive.ResolveModule("main"); err != nil || got.Path != "main.js" {
		t.Fatalf("ResolveModule(main) = %#v, %v", got, err)
	}
	if got, err := archive.ResolveModule("pkg"); err != nil || got.Path != "pkg/index.js" {
		t.Fatalf("ResolveModule(pkg) = %#v, %v", got, err)
	}
	if _, err := archive.ResolveModule("native/addon.node"); !errors.Is(err, ErrEntryUnpacked) {
		t.Fatalf("ResolveModule(unpacked) error = %v, want ErrEntryUnpacked", err)
	}
	if _, err := archive.ResolveModule("missing"); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("ResolveModule(missing) error = %v, want ErrEntryNotFound", err)
	}
}

func TestWriteAndOpenRealArchive(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "app.asar")
	if err := Write(archivePath, map[string][]byte{
		"main.js":      []byte("console.log('asar')"),
		"pkg/index.js": []byte("module.exports = 42"),
	}, map[string][]byte{
		"native/addon.node": []byte("native"),
	}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	archive, err := Open(archivePath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	main, err := archive.ReadFile("main.js")
	if err != nil {
		t.Fatalf("ReadFile(main.js) error = %v", err)
	}
	if string(main) != "console.log('asar')" {
		t.Fatalf("main.js = %q", main)
	}
	index, err := archive.ResolveModule("pkg")
	if err != nil {
		t.Fatalf("ResolveModule(pkg) error = %v", err)
	}
	if index.Path != "pkg/index.js" {
		t.Fatalf("ResolveModule(pkg) = %#v", index)
	}
	unpacked, err := archive.ReadFile("native/addon.node")
	if err != nil {
		t.Fatalf("ReadFile(unpacked) error = %v", err)
	}
	if string(unpacked) != "native" {
		t.Fatalf("unpacked = %q", unpacked)
	}
}

func TestNewArchiveRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		path    string
		entries []Entry
	}{
		{path: "app.zip"},
		{path: "app.asar", entries: []Entry{{Path: ""}}},
		{path: "app.asar", entries: []Entry{{Path: "bad", Size: -1}}},
	}
	for _, tc := range cases {
		if _, err := NewArchive(tc.path, tc.entries); !errors.Is(err, ErrInvalidPath) {
			t.Fatalf("NewArchive(%q, %#v) error = %v, want ErrInvalidPath", tc.path, tc.entries, err)
		}
	}
}
