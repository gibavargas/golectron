# Golectron

A Go-first attempt at an Electron-compatible desktop runtime.

## Thesis

Electron is heavy because every app ships a Chromium + Node.js stack. Golectron aims to keep the Electron developer contract while replacing the host/runtime with a small Go process and pluggable native webviews.

The honest target is staged compatibility:

1. **Main-process compatibility:** run common `main.js` Electron boot code: `app`, `BrowserWindow`, `ipcMain`, `dialog`.
2. **Renderer compatibility:** provide a preload bridge and `ipcRenderer` shim.
3. **Window backend:** attach the `BrowserWindow` abstraction to WebKitGTK/WebView2/WKWebView first, then optional Chromium/CDP when exact behavior is required.
4. **Packaging compatibility:** accept existing `package.json` app layout and build single binaries.
5. **Electron API coverage:** expand API-by-API using compatibility tests copied from real apps.

## Current status

This commit is the seed, not the finish line:

- Go CLI: `golectron run --dir examples/hello`
- Pure-Go JS runtime via `goja`
- Electron-like `require('electron')`
- Minimal CommonJS resolver for relative `.js`, `.json`, directory `index.*`, and local `node_modules` packages
- Minimal `app`, `BrowserWindow`, `ipcMain`, `ipcRenderer`, `dialog`
- `ipcMain.handle()` / `ipcRenderer.invoke()` preload bridge for request/response IPC
- `BrowserWindow.loadURL()` and `BrowserWindow.loadFile()` URL tracking
- Tests proving a basic Electron main process runs

## Quick start

```bash
go test ./...
go run ./cmd/golectron run --dir examples/hello
```

Expected output:

```text
hello from an Electron-shaped app
hello from preload IPC, Golectron
window[1] url=file:///path/to/golectron/examples/hello/renderer/index.html shown=true
```

## Compatibility contract

Unsupported APIs must fail loudly. Silent fake compatibility is worse than no compatibility because it creates production-only bugs.

## Why not claim 100% today?

A truly 100% compatible Electron replacement must match:

- Node.js module semantics
- Chromium/WebContents behavior
- IPC edge cases
- Native menus, trays, dialogs, shortcuts
- protocol/session/net APIs
- auto-update and packaging behavior
- platform-specific quirks across macOS, Windows, Linux

So the path is not a slogan. The path is a compatibility suite and a runtime that becomes more Electron-shaped every week.

## Roadmap

### P0 — useful MVP

- Load `package.json.main`
- Implement `app.whenReady`, `app.on`, `app.quit`
- Implement `BrowserWindow.loadURL`, `loadFile`, `show`, `hide`, `close`
- Implement `ipcMain` + `ipcRenderer` bridge
- Add native webview backend
- Package a demo app as one binary

### P1 — real app compatibility

- CommonJS resolver for local files, JSON modules, directory indexes, and `node_modules` packages
- `preload` support
- `webContents` events
- menu/tray/globalShortcut subset
- dialog subset
- session partition subset

### P2 — Electron test harness

- Import selected Electron fixtures
- Run compatibility tests against Electron and Golectron
- Publish coverage percentage by API namespace

## Non-goals for the first version

- Reimplementing all of Chromium
- Pretending every Electron app works before the compatibility suite says so
- Supporting native Node addons before pure JS apps work
