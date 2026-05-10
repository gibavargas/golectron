package desktop

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNormalizeOpenDialog(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	got, err := NormalizeOpenDialog(OpenDialogOptions{
		Title:       " Open ",
		DefaultPath: path,
		Properties:  []string{"openFile", "multiSelections"},
		Filters:     []FileFilter{{Name: "Images", Extensions: []string{"png", "jpg"}}},
	})
	if err != nil {
		t.Fatalf("NormalizeOpenDialog() error = %v", err)
	}
	if got.Title != "Open" || got.DefaultPath != path {
		t.Fatalf("normalized dialog = %#v", got)
	}
}

func TestNormalizeOpenDialogRejectsInvalidOptions(t *testing.T) {
	cases := []OpenDialogOptions{
		{DefaultPath: "relative"},
		{Properties: []string{"unknown"}},
		{Filters: []FileFilter{{Name: "", Extensions: []string{"txt"}}}},
		{Filters: []FileFilter{{Name: "Bad", Extensions: []string{".txt"}}}},
	}
	for _, tc := range cases {
		if _, err := NormalizeOpenDialog(tc); !errors.Is(err, ErrInvalidDialog) {
			t.Fatalf("NormalizeOpenDialog(%#v) error = %v, want ErrInvalidDialog", tc, err)
		}
	}
}

func TestShellEventsAndValidation(t *testing.T) {
	var shell Shell
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := shell.OpenExternal("https://example.test/path"); err != nil {
		t.Fatalf("OpenExternal() error = %v", err)
	}
	if err := shell.ShowItemInFolder(path); err != nil {
		t.Fatalf("ShowItemInFolder() error = %v", err)
	}
	if err := shell.TrashItem(path); err != nil {
		t.Fatalf("TrashItem() error = %v", err)
	}
	want := []string{
		"openExternal:https://example.test/path",
		"showItemInFolder:" + path,
		"trashItem:" + path,
	}
	if got := shell.Events(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Events() = %#v, want %#v", got, want)
	}
	if err := shell.OpenExternal("missing-scheme"); !errors.Is(err, ErrInvalidShell) {
		t.Fatalf("OpenExternal(invalid) error = %v, want ErrInvalidShell", err)
	}
	if message, err := shell.OpenPath(filepath.Join(t.TempDir(), "missing.txt")); err != nil || message == "" {
		t.Fatalf("OpenPath(missing) = %q, %v; want nonempty message and nil error", message, err)
	}
	if message, err := shell.OpenPath("relative"); !errors.Is(err, ErrInvalidShell) || message == "" {
		t.Fatalf("OpenPath(invalid) = %q, %v; want message and ErrInvalidShell", message, err)
	}
	if err := shell.ShowItemInFolder("relative"); !errors.Is(err, ErrInvalidShell) {
		t.Fatalf("ShowItemInFolder(invalid) error = %v, want ErrInvalidShell", err)
	}
}

func TestClipboardFormats(t *testing.T) {
	var clipboard Clipboard
	if _, err := clipboard.ReadText(); !errors.Is(err, ErrClipboardEmpty) {
		t.Fatalf("ReadText(empty) error = %v, want ErrClipboardEmpty", err)
	}
	clipboard.WriteText("plain")
	clipboard.WriteHTML("<b>html</b>")
	clipboard.WriteRTF(`{\rtf1}`)
	clipboard.WriteImage([]byte{1, 2, 3})
	if err := clipboard.WriteBuffer("electron-go/custom", []byte{4, 5, 6}); err != nil {
		t.Fatalf("WriteBuffer() error = %v", err)
	}
	if text, _ := clipboard.ReadText(); text != "plain" {
		t.Fatalf("ReadText() = %q", text)
	}
	if html, _ := clipboard.ReadHTML(); html != "<b>html</b>" {
		t.Fatalf("ReadHTML() = %q", html)
	}
	if rtf, _ := clipboard.ReadRTF(); rtf != `{\rtf1}` {
		t.Fatalf("ReadRTF() = %q", rtf)
	}
	image, err := clipboard.ReadImage()
	if err != nil {
		t.Fatalf("ReadImage() error = %v", err)
	}
	image[0] = 9
	again, _ := clipboard.ReadImage()
	if again[0] != 1 {
		t.Fatal("ReadImage() returned mutable clipboard image")
	}
	buffer, err := clipboard.ReadBuffer("electron-go/custom")
	if err != nil {
		t.Fatalf("ReadBuffer() error = %v", err)
	}
	buffer[0] = 9
	bufferAgain, _ := clipboard.ReadBuffer("electron-go/custom")
	if !reflect.DeepEqual(bufferAgain, []byte{4, 5, 6}) {
		t.Fatalf("ReadBuffer() = %#v, want defensive copy", bufferAgain)
	}
	if _, err := clipboard.ReadBuffer("missing"); !errors.Is(err, ErrClipboardEmpty) {
		t.Fatalf("ReadBuffer(missing) error = %v, want ErrClipboardEmpty", err)
	}
	if err := clipboard.WriteBuffer("", []byte{1}); !errors.Is(err, ErrInvalidDialog) {
		t.Fatalf("WriteBuffer(empty) error = %v, want ErrInvalidDialog", err)
	}
}

func TestCapturerSources(t *testing.T) {
	capturer, err := NewCapturer([]Source{
		{ID: "screen:1", Name: "Screen", Type: SourceScreen, Thumbnail: []byte{1}},
		{ID: "window:1", Name: "Window", Type: SourceWindow, Thumbnail: []byte{2}},
	})
	if err != nil {
		t.Fatalf("NewCapturer() error = %v", err)
	}
	screens := capturer.GetSources(SourceScreen)
	if len(screens) != 1 || screens[0].ID != "screen:1" {
		t.Fatalf("screens = %#v", screens)
	}
	screens[0].Thumbnail[0] = 9
	again := capturer.GetSources(SourceScreen)
	if again[0].Thumbnail[0] != 1 {
		t.Fatal("GetSources() returned mutable thumbnail")
	}
	if _, err := NewCapturer([]Source{{ID: "bad", Name: "Bad", Type: "audio"}}); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("NewCapturer(invalid) error = %v, want ErrInvalidSource", err)
	}
}
