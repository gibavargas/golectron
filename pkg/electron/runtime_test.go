package electron

import (
	"path/filepath"
	"testing"
)

func TestRunsElectronMainProcessSubset(t *testing.T) {
	dir := t.TempDir()
	script := `const { app, BrowserWindow } = require('electron')
app.on('ready', () => {
  const win = new BrowserWindow({ width: 800, height: 600 })
  win.loadURL('https://example.com')
  win.show()
})`

	r := New(dir)
	if _, err := r.vm.RunScript(filepath.Join(dir, "main.js"), script); err != nil {
		t.Fatalf("run script: %v", err)
	}
	if err := r.app.emit("ready"); err != nil {
		t.Fatalf("emit ready: %v", err)
	}

	windows := r.Windows()
	if len(windows) != 1 {
		t.Fatalf("expected 1 BrowserWindow, got %d", len(windows))
	}
	if windows[0].URL != "https://example.com" {
		t.Fatalf("url = %q", windows[0].URL)
	}
	if !windows[0].Shown {
		t.Fatalf("window should be shown")
	}
}

func TestUnsupportedRequireFailsLoudly(t *testing.T) {
	r := New(t.TempDir())
	_, err := r.vm.RunString(`require('fs')`)
	if err == nil {
		t.Fatalf("expected unsupported require to fail")
	}
}
