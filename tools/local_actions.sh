#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

export GOCACHE="${GOCACHE:-/private/tmp/electron-go-gocache}"

docker_bin="${DOCKER_BIN:-docker}"
docker_config="${DOCKER_CONFIG:-/private/tmp/electron-go-docker-config}"
docker_host="${DOCKER_HOST:-unix:///Users/jvguidi/.colima/default/docker.sock}"
host_arch="$(uname -m)"
if [[ -n "${DOCKER_PLATFORM:-}" ]]; then
  docker_platform="${DOCKER_PLATFORM}"
elif [[ "${host_arch}" == "arm64" || "${host_arch}" == "aarch64" ]]; then
  docker_platform="linux/arm64"
else
  docker_platform="linux/amd64"
fi
docker_image="${DOCKER_IMAGE:-golang:1.26-bookworm}"
cef_fetch_retries="${CEF_FETCH_RETRIES:-3}"
cef_fetch_timeout="${CEF_FETCH_TIMEOUT:-120s}"
if [[ -n "${CEF_PLATFORM:-}" ]]; then
  cef_platform="${CEF_PLATFORM}"
elif [[ "${docker_platform}" == "linux/arm64" ]]; then
  cef_platform="linuxarm64"
else
  cef_platform="linux64"
fi
case "${docker_platform}" in
  linux/amd64)
    docker_goarch="amd64"
    ;;
  linux/arm64)
    docker_goarch="arm64"
    ;;
  *)
    echo "unsupported DOCKER_PLATFORM=${docker_platform}" >&2
    exit 2
    ;;
esac

usage() {
  cat <<'USAGE'
Usage: tools/local_actions.sh <ci|goal-audit|benchmarks|conformance>

Runs the GitHub Actions gates locally so hosted Actions minutes are not used.
The Linux CEF benchmark/conformance jobs use linux64 on amd64 hosts and
linuxarm64 on arm64 hosts so local runs avoid hosted Actions minutes.
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
  COPYFILE_DISABLE=1 git ls-files -z | COPYFILE_DISABLE=1 tar --null -T - -cf local-actions-src.tar
  mkdir -p local-actions-out local-actions-cache
}

build_linux_helper_tools() {
  tools_dir="local-actions-out/tools/linux-${docker_goarch}"
  mkdir -p "${tools_dir}"
  CGO_ENABLED=0 GOOS=linux GOARCH="${docker_goarch}" go build -trimpath -o "${tools_dir}/benchmarks" ./tools/benchmarks
  CGO_ENABLED=0 GOOS=linux GOARCH="${docker_goarch}" go build -trimpath -o "${tools_dir}/cefresolve" ./tools/cefresolve
  CGO_ENABLED=0 GOOS=linux GOARCH="${docker_goarch}" go build -trimpath -o "${tools_dir}/ceffetch" ./tools/ceffetch
  CGO_ENABLED=0 GOOS=linux GOARCH="${docker_goarch}" go build -trimpath -o "${tools_dir}/memaudit" ./tools/memaudit
}

docker_run() {
  "${docker_bin}" --config "${docker_config}" -H "${docker_host}" run --rm \
    --platform "${docker_platform}" \
    -e CEF_PLATFORM="${cef_platform}" \
    -e CEF_FETCH_RETRIES="${cef_fetch_retries}" \
    -e CEF_FETCH_TIMEOUT="${cef_fetch_timeout}" \
    -v "${repo_root}/local-actions-src.tar:/src.tar:ro" \
    -v "${repo_root}/local-actions-out:/out" \
    -v "${repo_root}/local-actions-cache:/cache" \
    "${docker_image}" bash -lc "$1"
}

select_cef_manifest() {
  case "${CEF_PLATFORM}" in
    linux64)
      export CEF_MANIFEST=native/CEF_VERSION
      ;;
    linuxarm64)
      cp native/CEF_VERSION /tmp/CEF_VERSION.linuxarm64
      sed -i \
        -e "s/^PLATFORM=.*/PLATFORM=linuxarm64/" \
        -e "s/^ARCHIVE=.*/ARCHIVE=cef_binary_147.0.10+gd58e84d+chromium-147.0.7727.118_linuxarm64_minimal.tar.bz2/" \
        -e "s/^SHA1=.*/SHA1=b42ff1dc31a534161a7856889dbda17ac0f9670c/" \
        -e "s/^SHA256=.*/SHA256=UNRESOLVED/" \
        -e "s/^SIZE=.*/SIZE=381346274/" \
        /tmp/CEF_VERSION.linuxarm64
      export CEF_MANIFEST=/tmp/CEF_VERSION.linuxarm64
      ;;
    *)
      echo "unsupported CEF_PLATFORM=${CEF_PLATFORM}" >&2
      return 2
      ;;
  esac
}

install_node22() {
  case "$(uname -m)" in
    aarch64 | arm64)
      node_arch="arm64"
      ;;
    x86_64 | amd64)
      node_arch="x64"
      ;;
    *)
      echo "unsupported Node architecture: $(uname -m)" >&2
      return 2
      ;;
  esac
  node_version="v22.12.0"
  node_dir="node-${node_version}-linux-${node_arch}"
  curl -fsSL "https://nodejs.org/dist/${node_version}/${node_dir}.tar.xz" -o /tmp/node.tar.xz
  tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1
  node --version
  npm --version
}

install_electron_wrapper() {
  cat >/tmp/electron-go/electron-local <<'EOF'
#!/usr/bin/env bash
exec /tmp/electron-go/node_modules/.bin/electron --no-sandbox "$@"
EOF
  chmod +x /tmp/electron-go/electron-local
}

fetch_cef_with_retries() {
  attempts="${CEF_FETCH_RETRIES:-3}"
  timeout="${CEF_FETCH_TIMEOUT:-120s}"
  if [[ "${attempts}" -lt 1 ]]; then
    attempts=1
  fi
  for attempt in $(seq 1 "${attempts}"); do
    echo "[CEF] fetch attempt ${attempt}/${attempts} with timeout ${timeout}"
    if CEF_RESOLVE_TIMEOUT="${timeout}" CEF_FETCH_TIMEOUT="${timeout}" tools/fetch_cef.sh; then
      return 0
    fi
    if [[ "${attempt}" -lt "${attempts}" ]]; then
      sleep $((attempt * 5))
    fi
  done
  return 1
}

link_cached_cef_headers() {
  if [[ -z "${CEF_OUTPUT_DIR:-}" ]]; then
    return 0
  fi
  if [[ ! -e "${CEF_OUTPUT_DIR}/current/include/capi/cef_app_capi.h" ]]; then
    return 0
  fi
  mkdir -p native/cef
  ln -sfn "${CEF_OUTPUT_DIR}/current" native/cef/current
}

preflight_linux_toolchain() {
  write_source_archive
  build_linux_helper_tools
  if docker_run "$(declare -f select_cef_manifest)
set -euo pipefail
export PATH=/usr/local/go/bin:$PATH
mkdir -p /work
tar -xf /src.tar -C /work >/dev/null 2>&1
cd /work
go version
select_cef_manifest
go test ./tools/cefresolve -run \"^$\""; then
    return 0
  fi

  cat >&2 <<'EOF'
Local Linux Go preflight failed.

This host must be able to run Go 1.26 reliably for the selected Docker/CEF
platform. On Apple Silicon, linux/amd64 under Colima/QEMU may crash inside the
Go toolchain before the benchmark job reaches Electron-Go; use the default
linux/arm64 local path or set DOCKER_PLATFORM=linux/arm64 CEF_PLATFORM=linuxarm64.
EOF
  return 2
}

linux_common_preamble="$(declare -f select_cef_manifest)
$(declare -f install_node22)
$(declare -f install_electron_wrapper)
$(declare -f fetch_cef_with_retries)
$(declare -f link_cached_cef_headers)
set -euo pipefail
export PATH=/usr/local/go/bin:$PATH
export DEBIAN_FRONTEND=noninteractive
export CEF_OUTPUT_DIR=/cache/cef-${cef_platform}
export CEFRESOLVE_BIN=/out/tools/linux-${docker_goarch}/cefresolve
export CEFFETCH_BIN=/out/tools/linux-${docker_goarch}/ceffetch
export MEMAUDIT_BIN=/out/tools/linux-${docker_goarch}/memaudit
mkdir -p /work /out
tar -xf /src.tar -C /work >/dev/null 2>&1
cd /work
go version
select_cef_manifest
apt-get update -qq
apt-get install -y --no-install-recommends ca-certificates curl xz-utils time libgtk-3-0 libgdk-pixbuf2.0-0 libglib2.0-0 libnss3 libnspr4 libatk1.0-0 libatk-bridge2.0-0 libcups2 libdrm2 libxkbcommon0 libxcomposite1 libxdamage1 libxfixes3 libxrandr2 libgbm1 libasound2 xvfb xauth
install_node22
npm install --prefix /tmp/electron-go electron@42.0.0
install_electron_wrapper
fetch_cef_with_retries
link_cached_cef_headers
go build -tags electron_go_cef -o bin/electron-go ./cmd/electron-go"

run_benchmarks() {
  preflight_linux_toolchain
  docker_run "${linux_common_preamble}
	if ! timeout 45 env ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a /tmp/electron-go/electron-local compat/fixtures/benchmark-hello; then
		echo '[benchmark] official Electron smoke timed out or failed; continuing to measured benchmark for structured diagnostics' >&2
	fi
	if ! timeout 45 env ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a bin/electron-go compat/fixtures/benchmark-hello; then
		echo '[benchmark] Electron-Go smoke timed out or failed; continuing to measured benchmark for structured diagnostics' >&2
	fi
	mkdir -p /out/benchmark-artifacts
	timeout 210 env ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a /out/tools/linux-${docker_goarch}/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --run-order alternating --measure-rss --measure-process-tree-rss --output /out/benchmark-artifacts/hello-linux.json --electron /tmp/electron-go/electron-local --electron-go bin/electron-go --require-faster process_tree_rss_peak_median_kb
	timeout 210 env ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a /out/tools/linux-${docker_goarch}/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --run-order alternating --output /out/benchmark-artifacts/hello-linux-duration.json --electron /tmp/electron-go/electron-local --electron-go bin/electron-go
if ! timeout 210 env ELECTRON_GO_STARTUP_TRACE=1 ELECTRON_GO_ENABLE_SCOPED_MAIN_RUNNER=1 xvfb-run -a /out/tools/linux-${docker_goarch}/benchmarks --fixture ./compat/fixtures/benchmark-hello --iterations 5 --timeout 30s --output /out/benchmark-artifacts/electron-go-startup-trace-linux.json --electron-go bin/electron-go; then
	echo '[benchmark] startup trace diagnostic did not complete; main benchmark reports were already written' >&2
fi"
}

run_conformance() {
  preflight_linux_toolchain
  docker_run "${linux_common_preamble}
	timeout 45 xvfb-run -a /tmp/electron-go/electron-local compat/fixtures/benchmark-hello
	timeout 45 xvfb-run -a bin/electron-go compat/fixtures/benchmark-hello
	timeout 45 xvfb-run -a bin/electron-go --process-model-check compat/fixtures/benchmark-hello
	timeout 240 xvfb-run -a env ELECTRON_BIN=/tmp/electron-go/electron-local ELECTRON_GO_BIN=/work/bin/electron-go go test ./compat -timeout=180s -v"
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
