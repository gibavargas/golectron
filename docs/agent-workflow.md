# Agent Workflow

Electron-Go cannot reach full Electron parity through loose translation. Work
must be split into evidence-producing packets and merged only when each packet
improves the ledger, tests, or benchmark story.

## Packet Generation

Generate the next backlog from the compatibility ledger:

```sh
go run ./tools/workpackets --status unstarted,stubbed,partial --limit 10
go run ./tools/workpackets --area ipc --format json
go run ./tools/workpackets --id cef_bootstrap
```

Each packet names the Electron 42.0.0 behavior area, the current status, existing
evidence, and a ready-to-hand-off agent prompt.

## Worker Rules

Workers must own disjoint paths. A typical split is:

- compatibility inventory: `internal/compat`, `compat`, and compatibility docs;
- benchmark evidence: `tools/benchmarks` and benchmark docs;
- native bridge: `internal/native` and `native`;
- runtime behavior: `internal/runtime`, CLI wiring, and fixtures.

Workers must not mark ledger items `compatible` until tests prove parity against
official Electron on the same fixture.

## Translation Rules

The Go rewrite should not mechanically preserve Chromium's C++ inheritance
shape. New Go internals should prefer:

- composition over inheritance;
- typed structs over `any`;
- channels and goroutines for ownership transfer;
- value slices and zero-copy views where lifetimes are clear;
- `sync.Pool` only for hot allocation paths proven by benchmarks;
- explicit struct layout review for hot-path memory.

`unsafe` is allowed only at native ABI boundaries or when a benchmark proves the
tradeoff and tests cover the lifetime contract.

## Equivalence Loop

Every real porting packet should follow the same loop:

1. import or create an Electron fixture that exposes the behavior;
2. run it against official Electron and record the observable contract;
3. implement the Go/native path;
4. run the fixture against Electron-Go;
5. update ledger evidence only when results match.

Benchmarks are attached after correctness, not before.

## Runtime Safety Kernel

Bridge work must use `internal/native.Dispatcher` for CEF-to-Go handoff. New V8
or CEF callbacks should:

- copy borrowed CEF data before leaving the callback;
- enqueue a bounded dispatcher request;
- include browser id, frame id, origin, capability, method, and timeout;
- avoid blocking the CEF UI thread while Go handlers run;
- convert panics and policy failures into structured errors.

Do not expose raw IPC or dynamic payloads as the first bridge surface. The first
renderer bridge should prove one narrow capability call through the dispatcher
before wider Electron-shaped APIs are added.

## First Boss: `cef_bootstrap`

The first real browser packet is `cef_bootstrap`, not a broad Electron API.
It exists to prove the native architecture before higher-level parity work
depends on it.

Acceptance criteria:

- `runtime.LockOSThread` runs at executable entry before CEF or app init;
- CEF subprocess flags are routed before normal Electron-Go startup;
- one visible BrowserWindow loads `file://` content from the hello fixture;
- the CEF message loop owns the UI thread and exits on window close;
- renderer/GPU/network subprocesses do not remain as zombies;
- Linux x86_64 passes first, with macOS and Windows allowed to remain stubs;
- JavaScript execution, Electron IPC, and multi-window support stay out of scope.

Before implementing the CEF calls, run:

```sh
go run ./tools/cefresolve --json
tools/fetch_cef.sh
go build -tags electron_go_cef -o bin/electron-go ./cmd/electron-go
```
