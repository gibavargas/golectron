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

Run startup comparisons in alternating order to reduce fixed-order noise:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/hello \
  --iterations 10 \
  --timeout 30s \
  --run-order alternating \
  --electron "npx electron" \
  --electron-go "go run ./cmd/electron-go"
```

Run the CI-enforced hello comparison locally on Linux after staging CEF and
installing official Electron:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/benchmark-hello \
  --iterations 5 \
  --timeout 30s \
  --measure-rss \
  --measure-process-tree-rss \
  --output ./hello-linux.json \
  --electron /tmp/electron-go-bench/node_modules/.bin/electron \
  --electron-go ./bin/electron-go \
  --require-faster process_tree_rss_peak_median_kb
```

The `--require-faster` flag fails the run if any sample fails or if
Electron-Go does not beat official Electron for the named lower-is-better
metric. The GitHub Actions benchmark workflow uses this for the hello
process-tree RSS gate and uploads the raw JSON report.

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
- optional Linux process-tree peak RSS sampling through `/proc`;
- duration and RSS ratios when both `--electron` and `--electron-go` are present.
- optional CI gating with `--require-faster` for specific lower-is-better
  metrics.
- optional ratio gating with `--require-ratio metric=max-ratio`, for example
  `--require-ratio duration_median_ms=0.5` to enforce a 50% startup target.
- optional alternating sample order with `--run-order alternating`, recorded as
  `run_order: "alternating"` plus per-sample `sequence` and `pair` fields.
- optional fixture-level `benchmark-trace:` phase timings emitted by benchmark
  fixtures, such as app readiness, window creation, load start, and load finish,
  when the runtime executes the benchmark fixture main script and
  `ELECTRON_GO_BENCHMARK_TRACE=1` is set;
- median summaries for Electron-Go native startup traces and any emitted
  fixture-level benchmark traces.

It does not yet measure time to first visible paint, IPC latency, idle CPU,
package size, binary size, or build/package time. Do not cite those metrics from
this runner until they are implemented.

### Latest Linux Hello Result

PR benchmark run
`https://github.com/gibavargas/electron-go/actions/runs/25648091899` on commit
`67c895b` measured the `compat/fixtures/benchmark-hello` fixture with five
successful alternating-order iterations on Ubuntu 22.04:

- median wall-clock startup: Electron-Go `343ms`, Electron `470ms`
  (`electron_go_over_electron = 0.7298`, about `1.37x` faster);
- median process-tree peak RSS: Electron-Go `249388KB`, Electron `603308KB`
  (`electron_go_over_electron = 0.4134`, about `2.42x` lighter);
- median root max RSS: Electron-Go `248340KB`, Electron `210608KB`
  (`electron_go_over_electron = 1.1792`, Electron-Go is heavier for this
  process-only metric).
- Electron-Go native startup medians: `cef_initialize=127ms`,
  `create_browser=45ms`, `message_loop=103ms`, `cef_shutdown=26ms`,
  `total_native_start=302ms`.

Fixture-level phase traces are diagnostic and should be collected in explicit
trace runs. The normal headline benchmark keeps fixture tracing disabled to
avoid adding JavaScript and console-output work to only one side of the
comparison.

The previous fixed-order default run
`https://github.com/gibavargas/electron-go/actions/runs/25647836158` on commit
`9bae6c4` measured Electron-Go `411ms` vs Electron `543ms`. Prefer the
alternating-order run above for headline startup comparisons.

The previous traced run
`https://github.com/gibavargas/electron-go/actions/runs/25647216610` on commit
`6fd2e21` measured Electron-Go `380ms` vs Electron `508ms` and included official
Electron fixture trace medians. Treat those fixture trace medians as diagnostic
rather than headline performance evidence.

The earlier PR benchmark run
`https://github.com/gibavargas/electron-go/actions/runs/25645023440` on commit
`e4756dd` measured Electron-Go `339ms` vs Electron `476ms` before benchmark
phase-trace summaries were added. Keep comparing optimization candidates against
the current benchmark artifact shape rather than mixing pre-trace and post-trace
claims.

This is a measured startup improvement, but it is not yet the 50% faster target.
Keep tracking startup work against the raw benchmark artifact rather than
claiming the target is complete.

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
Prefer `--run-order alternating` for headline Electron vs Electron-Go startup
comparisons so both commands take turns running first within each pair.
Use `--require-ratio duration_median_ms=0.5` when a release gate must enforce
the claim that Electron-Go starts in no more than half the Electron median.

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
