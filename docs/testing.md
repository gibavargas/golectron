# Testing

Electron-Go uses three levels of tests.

## Go Unit Tests

Run:

```sh
go test ./...
```

These tests cover app metadata loading, runtime lifecycle, IPC routing, and the
compatibility ledger.

## Conformance Tests

Conformance tests compare Electron-Go against official Electron on the same
fixture applications.

Initial fixtures live in `compat/fixtures/hello`.

On macOS, official Electron GUI launches abort under the Codex seatbelt sandbox
before fixture code runs. The `compat` Go tests skip those official Electron
launches when `CODEX_SANDBOX` is set, and the JSON conformance runner marks the
fixture comparison as skipped instead of a mismatch. Run them outside the
sandbox for real e2e evidence. Set
`ELECTRON_GO_ALLOW_SANDBOXED_DARWIN_CONFORMANCE=1` only when you intentionally
want to reproduce the sandbox abort.

Run the JSON conformance runner with an installed official Electron binary:

```sh
go run ./tools/conformance \
  --fixture ./compat/fixtures/hello \
  --electron "electron" \
  --electron-go "go run ./cmd/electron-go" \
  --timeout 30s \
  --output conformance-report.json
```

Multiple fixtures can be passed by repeating `--fixture` or using comma-separated
values:

```sh
go run ./tools/conformance \
  --fixture ./compat/fixtures/hello,./compat/fixtures/another-fixture \
  --fixture ./compat/fixtures/yet-another-fixture \
  --electron "electron" \
  --electron-go "go run ./cmd/electron-go" \
  --timeout 30s \
  --output conformance-report.json
```

The command exits nonzero when official Electron and Electron-Go differ on exit
code, stdout, stderr, timeout status, or command error. To write evidence while
known gaps are still expected, keep the JSON report but allow the process to
exit zero:

```sh
go run ./tools/conformance \
  --fixture ./compat/fixtures/hello \
  --electron "electron" \
  --electron-go "go run ./cmd/electron-go" \
  --timeout 30s \
  --output conformance-report.json \
  --allow-mismatch
```

Required comparison points:

- process exit code;
- app lifecycle event order;
- `BrowserWindow` creation behavior;
- renderer load completion;
- preload execution;
- context isolation;
- sandbox behavior;
- IPC request and response behavior;
- console and error output.

The first scaffold cannot pass native conformance because the Chromium bridge is
not implemented yet. The expected Electron-Go result is the controlled native
bridge unavailable error.

## Release Gate

A release can claim full parity only when:

```sh
go test ./...
go run ./cmd/electron-go --check-parity
go run ./cmd/electron-go --runtime-parity-audit
go run ./cmd/electron-go --goal-audit
```

all pass, and the platform conformance suite passes against the latest stable
official Electron baseline.

`--check-parity` covers the 49-item compatibility ledger. It is not, by itself,
proof that every Electron application-runtime behavior is implemented. The
runtime fixture comparisons at the top of `compat/conformance_test.go` remain
explicitly gated until their underlying runtime features land:

- `TestHelloFixtureConformance` requires
  `ELECTRON_GO_ENABLE_IPC_CONFORMANCE=1` after IPC/preload parity is ready.
- `TestBenchmarkHelloFixtureConformance` requires
  `ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE=1` after native
  Chromium/Node/V8 main-process runtime parity is ready.

`--runtime-parity-audit` exits nonzero while either gate is skipped. Do not
claim full Electron app-runtime parity until it passes and the corresponding
fixture comparisons pass against official Electron.

`--goal-audit` combines the 49/49 ledger gate, e2e evidence audit, runtime
parity audit, and the current published startup benchmark ratio against the
`electron_go_over_electron <= 0.5` target. It is the final objective check and
must remain red until both full runtime parity and the 50% faster target are
proven.
