package electron

import (
	"os"
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

func TestCommonJSRequiresLocalJSAndJSONModules(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "config.json", `{"url":"https://example.test/from-json"}`)
	writeTestFile(t, dir, "window-factory.js", `
const config = require('./config.json')
exports.create = function(BrowserWindow) {
  const win = new BrowserWindow({ title: 'local module' })
  win.loadURL(config.url)
  return win
}
`)
	writeTestFile(t, dir, "main.js", `
const { app, BrowserWindow } = require('electron')
const factory = require('./window-factory')
app.on('ready', () => {
  const win = factory.create(BrowserWindow)
  win.show()
})
`)

	r := New(dir)
	if err := r.RunFile("main.js"); err != nil {
		t.Fatalf("run main.js: %v", err)
	}

	windows := r.Windows()
	if len(windows) != 1 {
		t.Fatalf("expected 1 BrowserWindow, got %d", len(windows))
	}
	if windows[0].URL != "https://example.test/from-json" {
		t.Fatalf("url = %q", windows[0].URL)
	}
	if !windows[0].Shown {
		t.Fatalf("window should be shown")
	}
}

func TestBrowserWindowLoadFileStoresFileURL(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", `<h1>Hello</h1>`)
	writeTestFile(t, dir, "main.js", `
const { app, BrowserWindow } = require('electron')
app.on('ready', () => {
  const win = new BrowserWindow()
  win.loadFile('index.html')
})
`)

	r := New(dir)
	if err := r.RunFile("main.js"); err != nil {
		t.Fatalf("run main.js: %v", err)
	}

	windows := r.Windows()
	if len(windows) != 1 {
		t.Fatalf("expected 1 BrowserWindow, got %d", len(windows))
	}
	want := "file://" + filepath.ToSlash(filepath.Join(dir, "index.html"))
	if windows[0].URL != want {
		t.Fatalf("url = %q, want %q", windows[0].URL, want)
	}
}

func TestPreloadCanInvokeIpcMainHandler(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "index.html", `<h1>IPC</h1>`)
	writeTestFile(t, dir, "preload.js", `
const { ipcRenderer } = require('electron')
ipcRenderer.invoke('double', 21).then(value => console.log('ipc:' + value))
`)
	writeTestFile(t, dir, "main.js", `
const { app, BrowserWindow, ipcMain } = require('electron')
ipcMain.handle('double', (_event, value) => value * 2)
app.on('ready', () => {
  const win = new BrowserWindow({ webPreferences: { preload: './preload.js' } })
  win.loadFile('index.html')
})
`)

	var lines []string
	r := New(dir, WithStdout(func(s string) { lines = append(lines, s) }))
	if err := r.RunFile("main.js"); err != nil {
		t.Fatalf("run main.js: %v", err)
	}

	if !containsLine(lines, "ipc:42") {
		t.Fatalf("expected preload IPC output, got %#v", lines)
	}
}

func containsLine(lines []string, want string) bool {
	for _, line := range lines {
		if line == want {
			return true
		}
	}
	return false
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
