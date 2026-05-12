#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

export GOCACHE="${GOCACHE:-/private/tmp/electron-go-gocache}"

docker_bin="${DOCKER_BIN:-docker}"
docker_config="${DOCKER_CONFIG:-/private/tmp/electron-go-docker-config}"
docker_host="${DOCKER_HOST:-unix:///Users/jvguidi/.colima/default/docker.sock}"
docker_platform="${DOCKER_PLATFORM:-linux/amd64}"
docker_image="${DOCKER_IMAGE:-golang:1.26-bookworm}"

usage() {
  cat <<'USAGE'
Usage: tools/local_actions.sh <ci|goal-audit|benchmarks|conformance>

Runs the GitHub Actions gates locally so hosted Actions minutes are not used.
The Linux CEF benchmark/conformance jobs require linux/amd64 because the staged
CEF bundle is linux64.
USAGE
}

run_ci() {
  go test ./...
  go run ./tools/cefresolve --json || true
  go run ./tools/memaudit --root . --check-cef-layout ./bin || true
  go run ./cmd/electron-go --check-parity || true
}

run_goal_audit() {
  ELECTRON_GO_ENABLE_IPC_CONFORMANCE=1 \
    ELECTRON_GO_ENABLE_RUNTIME_FIXTURE_CONFORMANCE=1 \
    go run ./cmd/electron-go --goal-audit
}

write_source_archive() {
  git ls-files -z | tar --null -T - -cf local-actions-src.tar
  mkdir -p local-actions-out
}

docker_run() {
  "${docker_bin}" --config "${docker_config}" -H "${docker_host}" run --rm \
    --platform "${docker_platform}" \
    -v "${repo_root}/local-actions-src.tar:/src.tar:ro" \
    -v "${repo_root}/local-actions-out:/out" \
    "${docker_image}" bash -lc "$1"
}

preflight_linux_toolchain() {
  write_source_archive
  if docker_run 'set -euo pipefail
export PATH=/usr/local/go/bin:$PATH
mkdir -p /work
tar -xf /src.tar -C /work >/dev/null 2>&1
cd /work
go version
go test ./tools/cefresolve -run "^$"'; then
    return 0
  fi

  cat >&2 <<'EOF'
Local linux/amd64 Go preflight failed.

This host must be able to run Go 1.26 linux/amd64 reliably because the staged
CEF bundle is linux64. On Apple Silicon, Colima/QEMU may crash inside the Go
toolchain before the benchmark job reaches Electron-Go.
EOF
  return 2
}

linux_common_preamble='set -euo pipefail
export PATH=/usr/local/go/bin:$PATH
export DEBIAN_FRONTEND=noninteractive
mkdir -p /work /out
tar -xf /src.tar -C /work >/dev/null 2>&1
cd /work
go version
apt-get update -qq
apt-get install -y --no-install-recommends libgtk-3-0 libgdk-pixbuf2.0-0 libglib2.0-0 libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2 xvfb npm
npm install --prefix /tmp/electron-go electron@42.0.0
tools/fetch_cef.sh
go build -tags electron_go_cef -o bin/electron-go ./cmd/electron-go'

run_benchmarks() {
  preflight_linux_toolchain
  docker_run "${linux_common_preamble}
ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a timeout 30 /tmp/electron-go/node_modules/.bin/electron compat/fixtures/benchmark-hello
ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a timeout 30 bin/electron-go compat/fixtures/benchmark-hello
mkdir -p /out/benchmark-artifacts
ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a go run ./tools/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --run-order alternating --measure-rss --measure-process-tree-rss --output /out/benchmark-artifacts/hello-linux.json --electron /tmp/electron-go/node_modules/.bin/electron --electron-go bin/electron-go --require-faster process_tree_rss_peak_median_kb
ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a go run ./tools/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --run-order alternating --output /out/benchmark-artifacts/hello-linux-duration.json --electron /tmp/electron-go/node_modules/.bin/electron --electron-go bin/electron-go
ELECTRON_GO_STARTUP_TRACE=1 ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a go run ./tools/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --output /out/benchmark-artifacts/electron-go-startup-trace-linux.json --electron-go bin/electron-go"
}

run_conformance() {
  preflight_linux_toolchain
  docker_run "${linux_common_preamble}
xvfb-run -a timeout 30 /tmp/electron-go/node_modules/.bin/electron compat/fixtures/benchmark-hello
xvfb-run -a timeout 30 bin/electron-go compat/fixtures/benchmark-hello
xvfb-run -a timeout 30 bin/electron-go --process-model-check compat/fixtures/benchmark-hello
xvfb-run -a env ELECTRON_BIN=/tmp/electron-go/node_modules/.bin/electron ELECTRON_GO_BIN=/work/bin/electron-go go test ./compat"
}

case "${1:-}" in
  ci)
    run_ci
    ;;
  goal-audit)
    run_goal_audit
    ;;
  benchmarks)
    run_benchmarks
    ;;
  conformance)
    run_conformance
    ;;
  *)
    usage
    exit 2
    ;;
esac
