# Memory Strategy

Electron-Go's performance goal is not a slogan. Memory claims must come from
benchmarks, and memory-sensitive code must stay visible to review.

## Audit

Run:

```sh
go run ./tools/memaudit --root .
go run ./tools/memaudit --root . --json
go run ./tools/memaudit --root . --check-cef-layout ./bin
```

The audit reports:

- `any` uses;
- `interface{}` uses;
- `unsafe` imports;
- `sync.Pool` uses.
- CEF runtime layout when `--check-cef-layout` is provided.

These are not automatically wrong. IPC payloads, native ABI boundaries, and
measured allocation hot paths can justify them. The point is to force each one
to be intentional.

## Rules

- Prefer typed structs and generics over `any`.
- Keep `unsafe` at native ABI boundaries unless a benchmark proves otherwise.
- Use `sync.Pool` only after a benchmark identifies allocation churn.
- Prefer slices and views over copying byte buffers when lifetimes are clear.
- Copy borrowed native callback buffers before they outlive the CEF callback;
  use dispatcher `BorrowPayload` only when ownership is proven by tests.
- Review hot structs for padding after a benchmark shows they matter.

## Release Claims

No README, release note, or benchmark document may claim RAM savings until the
benchmark runner compares official Electron and Electron-Go on the same fixture,
same machine, and same Electron baseline.
