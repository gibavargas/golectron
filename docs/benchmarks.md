# Benchmarks

Benchmark claims must be reproducible and measured against official Electron on
the same machine. Do not write "Electron-Go is faster" unless a checked-in or
published benchmark report proves the specific metric.

## End-To-End Runner

The benchmark runner launches the same fixture with official Electron and
Electron-Go, repeats each command, and emits JSON with raw samples, summaries,
and ratios when both runs are present.

Run a timing-only comparison:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/hello \
  --iterations 10 \
  --timeout 30s \
  --electron "npx electron" \
  --electron-go "go run ./cmd/electron-go"
```

Run timing plus max RSS collection on macOS or Linux:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/hello \
  --iterations 10 \
  --timeout 30s \
  --measure-rss \
  --output ./benchmarks-local.json \
  --electron "npx electron" \
  --electron-go "go run ./cmd/electron-go"
```

Use quoted commands when a command argument contains spaces:

```sh
go run ./tools/benchmarks \
  --fixture "./compat/fixtures/hello world" \
  --electron "npx electron --no-sandbox" \
  --electron-go "go run ./cmd/electron-go --some-flag='two words'"
```

The runner appends `--fixture` as the final argument to each command. The command
parser supports single quotes, double quotes, escaped spaces, and empty quoted
arguments; it is not a full shell and does not expand variables, globs, pipes, or
redirections.

## Current Evidence

The runner currently records:

- per-iteration wall-clock duration;
- per-iteration exit code, timeout state, and error output on failure;
- min, median, mean, and max duration for successful runs;
- optional max resident set size on macOS via `/usr/bin/time -l`;
- optional max resident set size on Linux via `/usr/bin/time -v`;
- duration and RSS ratios when both `--electron` and `--electron-go` are present.

It does not yet measure time to first window, IPC latency, window creation time,
navigation/load time, idle CPU, package size, binary size, or build/package time.
Do not cite those metrics from this runner until they are implemented.

## Result Format

By default, JSON is written to stdout. With `--output`, JSON is written to the
specified file and the runner prints the path to stderr.

Ratios use lower-is-better metrics:

- `electron_go_over_electron`: values below `1.0` mean Electron-Go used less time
  or memory for that metric.
- `electron_over_electron_go`: values above `1.0` mean Electron was that many
  times slower or heavier than Electron-Go for that metric.

## Honesty Rules

Published results must include:

- Electron version and install source;
- Electron-Go commit SHA;
- OS, CPU, memory, and power state;
- whether the first run was cold or excluded as warmup;
- exact benchmark command;
- raw JSON report;
- metric-specific ratio from the report.

Compare medians across multiple successful iterations. If any sample failed or
timed out, either fix the benchmark setup or disclose the failures next to the
claim. Do not mix machines, OS versions, power modes, fixtures, or commands in a
single ratio.

Acceptable claim:

```text
On macOS arm64, fixture ./compat/fixtures/hello, 10 successful iterations:
Electron-Go median wall-clock launch was 4.00x faster than official Electron
using tools/benchmarks report benchmarks-local.json.
```

Unacceptable claim:

```text
Electron-Go is faster and lighter than Electron.
```

If a benchmark is slower or heavier, track it as a performance issue unless the
slowdown is explicitly accepted.

## Go Microbenchmarks

Run:

```sh
go test -run '^$' -bench . -benchmem ./...
```

These measure internal runtime overhead and are not substitutes for full
official Electron comparison benchmarks.
