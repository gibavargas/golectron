# Agent Workflow

Electron-Go cannot reach full Electron parity through loose translation. Work
must be split into evidence-producing packets and merged only when each packet
improves the ledger, tests, or benchmark story.

## Packet Generation

Generate the next backlog from the compatibility ledger:

```sh
go run ./tools/workpackets --status unstarted,stubbed,partial --limit 10
go run ./tools/workpackets --area ipc --format json
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
