# Electron-Go

Electron-Go is a Go-led compatibility runtime for Electron applications. It
tracks Electron API behavior in a machine-checkable ledger, validates selected
behavior against official Electron fixtures, and ships reproducible benchmark
tooling for comparing Electron-Go with upstream Electron on the same host.

This repository targets Electron `42.0.0`:

- Chromium `148.0.7778.96`
- Node.js `24.15.0`
- V8 `14.8.178.14` (`process.versions.v8` reports
  `14.8.178.14-electron.0`)
- Native module ABI `146`

Electron-Go is published under the MIT License. See [LICENSE](LICENSE).

## Project Results

Current audited project accomplishments:

- The Electron compatibility ledger is complete at `49/49` compatible entries.
- The e2e evidence audit has no missing evidence entries.
- The runtime audit has opt-in fixture gates for IPC/preload parity and the
  scoped native Chromium runtime fixture.
- The benchmark harness compares official Electron and Electron-Go on the same
  fixture, records raw JSON artifacts, supports alternating run order, process
  tree RSS sampling, ratio gates, and Electron-Go native startup traces.
- The CEF bootstrap tooling resolves, downloads, verifies, extracts, and stages
  Chromium Embedded Framework runtime assets for Linux.
- The local Actions substitute can run the same CI, goal-audit, conformance,
  and benchmark entry points without consuming hosted GitHub Actions minutes.
- The macOS/Apple Silicon local benchmark path can select a native
  `linux/arm64` Docker container and matching `linuxarm64` CEF artifact instead
  of relying on flaky `linux/amd64` emulation.

Latest published benchmark evidence is from GitHub Actions run
`25707723412` on commit `b1ae4e1`:

- Median startup with RSS probes: Electron-Go `393ms`, Electron `624ms`
  (`electron_go_over_electron = 0.6298`, about `1.59x` faster).
- Median startup without RSS probes: Electron-Go `390ms`, Electron `520ms`
  (`electron_go_over_electron = 0.7500`, about `1.33x` faster).
- Median process-tree peak RSS: Electron-Go `251284KB`, Electron `606900KB`
  (`electron_go_over_electron = 0.4140`, about `2.42x` lighter).
- Electron-Go native startup trace medians: `cef_initialize=144ms`,
  `window_created=199ms`, `did_finish_load=328ms`,
  `total_native_start=329ms`.

The final stated project goal is not yet complete. The compatibility ledger is
green, but the published startup ratio must reach
`electron_go_over_electron <= 0.5` before the project can claim "50% faster than
Electron" for startup with full feature parity.

## What Is Implemented

The repository includes:

- `cmd/electron-go`: the Electron-Go command-line entry point and parity audit
  commands.
- `internal/compat/ledger.json`: the 49-item compatibility ledger and target
  Electron baseline.
- `compat/fixtures`: official-Electron comparison fixtures.
- `internal/mainrunner`: scoped Electron-style main-process execution for the
  benchmark fixture shape.
- `internal/native`: native bridge abstractions plus CEF-backed Linux runtime
  support behind the `electron_go_cef` build tag.
- `tools/cefresolve` and `tools/ceffetch`: CEF artifact resolution, checksum
  verification, extraction, and runtime staging.
- `tools/benchmarks`: reproducible benchmark runner for timing, RSS, process
  tree RSS, alternating-order sampling, and ratio gates.
- `tools/local_actions.sh`: local substitute for the GitHub Actions workflows.

## Quick Start

Run the normal Go test suite:

```sh
go test ./...
```

Print the current compatibility state:

```sh
go run ./cmd/electron-go --compat-json
go run ./cmd/electron-go --check-parity
```

Run the full objective audit:

```sh
ELECTRON_GO_ENABLE_IPC_CONFORMANCE=1 \
ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE=1 \
go run ./cmd/electron-go --goal-audit
```

At the time of this README update, that audit still fails the performance
target because the latest published ratio is `0.6298`, not `<= 0.5`.

## Local Actions

Hosted GitHub Actions minutes are not required for local validation. Use:

```sh
make local-ci
make local-goal-audit
make local-benchmarks
```

or directly:

```sh
tools/local_actions.sh ci
tools/local_actions.sh goal-audit
tools/local_actions.sh conformance
tools/local_actions.sh benchmarks
```

On Apple Silicon, `tools/local_actions.sh` defaults to `linux/arm64` Docker and
`linuxarm64` CEF. Override when needed:

```sh
DOCKER_PLATFORM=linux/amd64 CEF_PLATFORM=linux64 tools/local_actions.sh benchmarks
```

The local benchmark runner installs Node `22.12.0` inside the container because
Electron `42.0.0` requires Node `>=22.12.0` for its installer.

## CEF Bootstrap

Stage the CEF runtime layout:

```sh
tools/fetch_cef.sh
```

The default manifest is [native/CEF_VERSION](native/CEF_VERSION). A different
manifest can be selected with `CEF_MANIFEST`:

```sh
CEF_MANIFEST=/tmp/CEF_VERSION.linuxarm64 tools/fetch_cef.sh
```

`tools/ceffetch` verifies available upstream checksums and stages `bin/libcef.so`
plus required runtime resources.

## Benchmarks

Run the benchmark harness after staging CEF and installing official Electron:

```sh
go run ./tools/benchmarks \
  --fixture ./compat/fixtures/benchmark-hello \
  --iterations 5 \
  --timeout 30s \
  --run-order alternating \
  --electron /tmp/electron-go/node_modules/.bin/electron \
  --electron-go ./bin/electron-go \
  --require-ratio duration_median_ms=0.5
```

The benchmark claim is intentionally narrow: it applies only to the measured
fixture, platform, commands, and artifacts. Do not generalize a benchmark result
to arbitrary Electron apps without matching conformance and benchmark evidence.

## Release And Parity Rules

A release may claim full Electron compatibility only when all of these pass:

```sh
go test ./...
go run ./cmd/electron-go --check-parity
go run ./cmd/electron-go --runtime-parity-audit
go run ./cmd/electron-go --goal-audit
```

`--check-parity` covers the 49-item ledger. `--goal-audit` is stricter: it
combines ledger completion, e2e evidence, runtime fixture gates, and the current
published performance ratio.

## Documentation

See:

- [Compatibility](docs/compatibility.md)
- [Testing](docs/testing.md)
- [Benchmarks](docs/benchmarks.md)
- [Architecture](docs/architecture.md)
- [Agent Workflow](docs/agent-workflow.md)
- [Memory Strategy](docs/memory-strategy.md)
- [CEF Bootstrap](docs/cef-bootstrap.md)

## Attribution

Electron-Go targets compatibility with Electron and follows Electron public
behavior as the reference contract. Electron is distributed under the MIT
License by the Electron contributors.
