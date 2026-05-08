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
