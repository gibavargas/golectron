# Benchmarks

Benchmark claims must be reproducible and measured against official Electron on
the same machine.

Do not write "Electron-Go is faster" unless a benchmark result proves it. Use
ratios only where the data supports the claim.

## Go Microbenchmarks

Run:

```sh
go test -run '^$' -bench . -benchmem ./...
```

These measure internal runtime overhead and are not substitutes for full Electron
comparison benchmarks.

## End-To-End Benchmarks

Use the benchmark runner:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/hello \
  --electron "npx electron" \
  --electron-go "go run ./cmd/electron-go"
```

The runner records:

- cold start;
- warm start;
- time to first window;
- IPC latency;
- window creation time;
- navigation/load time;
- memory RSS;
- idle CPU;
- package size;
- binary size;
- build/package time.

The current scaffold records command timing and exit status only. Native runtime
milestones must extend it with first-window, IPC, memory, CPU, and package
metrics before any speed claim is valid.

## Result Format

Published results must include:

- Electron version;
- Electron-Go commit SHA;
- OS, CPU, memory, and power state;
- exact commands;
- raw measurements;
- computed ratio.

Example language after measurement:

```text
Cold start: Electron-Go 1.32x faster than Electron on macOS 15 arm64.
```

If a benchmark is slower, track it as a performance issue unless the slowdown is
explicitly accepted.
