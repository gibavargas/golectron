# Electron-Go

Electron-Go is a Go-led, bundled-Chromium compatibility runtime for Electron
applications. The project goal is full feature parity with the latest stable
Electron release, measured by conformance tests, a compatibility ledger, and
reproducible benchmarks.

The current compatibility baseline is Electron 42.0.0:

- Chromium 148.0.7778.96
- Node.js 24.15.0
- V8 14.8.178.14 (`process.versions.v8` reports `14.8.178.14-electron.0`)
- Native module ABI 146

Electron-Go is not considered complete until every public Electron API and
supported platform behavior is marked `compatible` with evidence in the ledger.

## Current State

This repository is the first implementation scaffold. It includes:

- a runnable `electron-go` CLI;
- app metadata loading from Electron-style `package.json` files;
- a runtime lifecycle with a controlled native-bridge boundary;
- an IPC router used by future JS bindings;
- a compatibility ledger for the parity gate;
- deterministic CEF fetch/link/layout tooling for the first Linux bootstrap;
- conformance fixtures and benchmark documentation.

The native Chromium/Node/V8 bridge is intentionally unavailable in this first
commit. The CLI reports that state explicitly instead of pretending to run an
Electron app.

## Quick Start

```sh
go test ./...
go run ./cmd/electron-go --hello
```

The second command should exit with a native bridge unavailable message until
the bundled Chromium bridge is implemented.

## Parity Rule

No release may claim full Electron compatibility unless:

1. the latest stable Electron baseline has been refreshed;
2. every ledger item is `compatible`;
3. conformance tests pass against both official Electron and Electron-Go;
4. benchmark results are published with the exact commands and host details.

See:

- [Compatibility](docs/compatibility.md)
- [Testing](docs/testing.md)
- [Benchmarks](docs/benchmarks.md)
- [Architecture](docs/architecture.md)
- [Agent Workflow](docs/agent-workflow.md)
- [Memory Strategy](docs/memory-strategy.md)
- [CEF Bootstrap](docs/cef-bootstrap.md)

## Attribution

Electron-Go targets compatibility with Electron and follows its public behavior
as the reference contract. Electron is distributed under the MIT License by the
Electron contributors.
