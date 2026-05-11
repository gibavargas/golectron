# Benchmarks

Benchmark claims must be reproducible and measured against official Electron on
the same machine. Do not write "Electron-Go is faster" unless a checked-in or
published benchmark report proves the specific metric.

## End-To-End Runner

The benchmark runner launches the same fixture with official Electron and
Electron-Go, repeats each command, and emits JSON with raw samples, summaries,
and ratios when both runs are present.

The scoped CEF-backed Electron-Go app path recognizes the benchmark fixture
`main.js` lifecycle and models `app.whenReady()`, `BrowserWindow`,
`loadFile()`, `did-finish-load`, close, and `app.quit()` around a native CEF
window. This is stronger than the earlier direct `index.html` bootstrap, but it
is still a constrained fixture runner rather than a general Node/V8
main-process implementation. Treat the headline number as scoped
Electron-style app startup evidence, not proof of arbitrary Electron
main-process app parity.

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

Run a warm Electron-Go window-start probe for a recognized Electron-style app:

```sh
bin/electron-go \
  --warm-benchmark-check 1 \
  ./compat/fixtures/benchmark-hello
```

This initializes CEF before timing the recognized `main.js` lifecycle and emits
JSON for the single visible-window launch. It is an architecture probe for
separating CEF initialization cost from window lifecycle cost. Repeated warm
windows currently require a different CEF message-loop architecture and are
rejected explicitly. Do not mix this with the cold process-start headline result
or the `cmd/electron-go --goal-audit` performance gate until there is a matching
official-Electron baseline and explicit product decision to target warm starts.

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
- paired comparison summaries for alternating runs, including the median
  per-pair `electron_go_over_electron` ratio for duration and memory metrics.
- optional fixture-level `benchmark-trace:` phase timings emitted by benchmark
  fixtures, such as app readiness, window creation, load start, and load finish,
  when the runtime executes the benchmark fixture main script and
  `ELECTRON_GO_BENCHMARK_TRACE=1` is set;
- median summaries for Electron-Go native startup traces and any emitted
  fixture-level benchmark traces. Startup traces are diagnostic and should be
  captured separately from headline timing comparisons because
  `ELECTRON_GO_STARTUP_TRACE=1` instruments only Electron-Go.

It does not yet measure time to first visible paint, IPC latency, idle CPU,
package size, binary size, or build/package time. Do not cite those metrics from
this runner until they are implemented.

### Latest Linux Hello Result

PR benchmark run
`https://github.com/gibavargas/electron-go/actions/runs/25672475992` on commit
`f690493` measured the scoped runtime path for the
`compat/fixtures/benchmark-hello` fixture with five successful
alternating-order iterations on Ubuntu 22.04:

- median wall-clock startup: Electron-Go `415ms`, Electron `534ms`
  (`electron_go_over_electron = 0.7772`, about `1.29x` faster);
- median paired wall-clock startup ratio:
  `electron_go_over_electron_median = 0.7798` across five successful pairs;
- median process-tree peak RSS: Electron-Go `248960KB`, Electron `587572KB`
  (`electron_go_over_electron = 0.4237`, about `2.36x` lighter);
- paired process-tree peak RSS ratio:
  `electron_go_over_electron_median = 0.4230` across five successful pairs;
- median root max RSS: Electron-Go `247808KB`, Electron `212488KB`
  (`electron_go_over_electron = 1.1662`, Electron-Go is heavier for this
  process-only metric).
- separate Electron-Go scoped startup-trace medians: `cef_initialize=148ms`,
  `app_ready=148ms`, `window_created=199ms`, `load_start=330ms`,
  `did_finish_load=330ms`, `quit_requested=330ms`, `cef_shutdown=32ms`,
  `total_native_start=364ms`.

Newer Electron-Go startup traces may also include diagnostic message-loop split
fields such as `message_loop_to_load_end`, `load_end_to_before_close`, and
`before_close_to_loop_return`. These are Electron-Go-only diagnostics for
finding where `message_loop` time is spent; do not use them as headline
Electron-vs-Electron-Go comparison metrics.

Fixture-level phase traces and Electron-Go native startup traces are diagnostic
and should be collected in explicit trace runs. The normal headline benchmark
keeps runtime-specific tracing disabled to avoid adding console-output work to
only one side of the comparison.

The previous alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25671806700` on commit
`3de47d2` measured Electron-Go `362ms` vs Electron `468ms` before defaulting the
recognized main-runner path.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25671322163` on commit
`e724376` measured Electron-Go `415ms` vs Electron `546ms` before avoiding
duplicate CEF startup switch appends.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25669931890` on commit
`c5fd858` measured Electron-Go `427ms` vs Electron `578ms` before adding scoped
native-phase startup trace evidence.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25669399901` on commit
`b7a5521` measured Electron-Go `428ms` vs Electron `560ms` before creating the
scoped benchmark window directly at the target URL.

Manual conformance run
`https://github.com/gibavargas/electron-go/actions/runs/25670364754` revalidated
the current branch with the opt-in scoped main-runner probe, scoped benchmark
fixture conformance gate, scoped hello IPC/preload conformance gate, CEF
runtime-process-model gate, and the full Go conformance suite.

Manual conformance run
`https://github.com/gibavargas/electron-go/actions/runs/25673972317` on commit
`d896d4d` also ran the warm scoped benchmark probe. It measured CEF
initialization at `127ms`, one recognized benchmark-window lifecycle at `165ms`,
and shutdown at `24ms`. Treat this as architecture evidence for separating
runtime initialization from window startup, not as the cold-start headline
metric. The artifact may include headless Chromium diagnostic output before the
JSON payload.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25649084479` on commit
`46be416` measured Electron-Go `405ms` vs Electron `532ms` before adding the
normal-launch fast path and static CEF callback handlers. The newer run above is
the current headline artifact; it shows the cleanup did not move startup toward
the 50% faster target.

After downloading a published benchmark artifact, update the embedded goal audit
evidence with:

```sh
go run ./tools/goalevidence \
  --benchmark ./benchmark-artifacts/hello-linux.json \
  --run-url https://github.com/gibavargas/electron-go/actions/runs/<run-id> \
  --commit <measured-commit> \
  --output ./cmd/electron-go/goal_evidence.json
```

The previous alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25648847257` on commit
`9cccf56` measured Electron-Go `420ms` vs Electron `568ms` and added paired
benchmark ratio reporting.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25648608061` on commit
`b85dbcd` measured Electron-Go `347ms` vs Electron `426ms` after quieting normal
Electron-Go runtime status output.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25648373123` on commit
`75e0f25` measured Electron-Go `437ms` vs Electron `570ms` after startup traces
were split into a separate diagnostic artifact.

The earlier alternating-order run
`https://github.com/gibavargas/electron-go/actions/runs/25648091899` on commit
`67c895b` measured Electron-Go `343ms` vs Electron `470ms`. Treat this as prior
evidence; the current headline above is the latest uploaded benchmark artifact.

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
When alternating runs are available, inspect both the aggregate median
comparison and the paired median ratio before making a startup claim.
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
