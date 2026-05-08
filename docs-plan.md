# Golectron Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Build a Go runtime that can run increasingly large subsets of Electron apps faster and with smaller memory footprint.

**Architecture:** A Go host owns lifecycle, windows, IPC, packaging, and native integrations. JavaScript main-process code runs in a controlled JS engine with an Electron-compatible module shim. Renderer windows use a pluggable backend: native webview for speed/small footprint, Chromium/CDP fallback for strict compatibility.

**Tech Stack:** Go 1.22+, goja for main-process JS, native webview backend, compatibility tests, package.json app layout.

---

## Phase 0: Seed runtime

### Task 1: CLI app runner

**Objective:** Run `package.json.main` from an app directory.

**Files:**
- Create: `cmd/golectron/main.go`
- Test: manual with `examples/hello`

**Verification:**

```bash
go run ./cmd/golectron run --dir examples/hello
```

Expected: one synthetic window is created and printed.

### Task 2: Electron module shim

**Objective:** Support `const { app, BrowserWindow } = require('electron')`.

**Files:**
- Create: `pkg/electron/runtime.go`
- Test: `pkg/electron/runtime_test.go`

**Verification:**

```bash
go test ./pkg/electron
```

Expected: tests pass.

## Phase 1: Make it useful

### Task 3: CommonJS resolver

**Objective:** Support local `require('./file')`, JSON modules, and basic `node_modules` lookup.

**Acceptance:** A two-file Electron main process runs unchanged.

### Task 4: BrowserWindow lifecycle

**Objective:** Implement `loadFile`, `close`, `destroy`, `isDestroyed`, `setTitle`, `getTitle`.

**Acceptance:** Lifecycle tests pass and unsupported methods remain explicit.

### Task 5: IPC bridge

**Objective:** Implement `ipcMain.handle`, `ipcMain.on`, `ipcRenderer.invoke`, `ipcRenderer.send`.

**Acceptance:** Renderer can invoke main and receive response.

### Task 6: Native webview backend

**Objective:** Replace synthetic windows with a real window backend.

**Acceptance:** `examples/hello` opens a real native window on Linux.

## Phase 2: Real Electron compatibility

### Task 7: Compatibility test suite

**Objective:** Create fixtures that run both on Electron and Golectron.

**Acceptance:** CI reports pass/fail per Electron API namespace.

### Task 8: Package single binary

**Objective:** Embed app resources and ship one binary.

**Acceptance:** `golectron package examples/hello` outputs a runnable app binary.

## Phase 3: Performance proof

### Task 9: Benchmark against Electron

**Objective:** Measure cold start, RSS, binary size, and window creation time.

**Acceptance:** README contains reproducible benchmark commands and results.
