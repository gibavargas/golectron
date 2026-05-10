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
```

both pass, and the platform conformance suite passes against the latest stable
official Electron baseline.
