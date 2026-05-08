#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

CEF_VERSION="147.0.10+gd58e84d+chromium-147.0.7727.118"
CEF_TARBALL="cef_binary_${CEF_VERSION}_linux64_minimal.tar.bz2"
CEF_SHA256="087ccfa1f0438429e20d91293b15cae08ee9e7b4988cc018d47568dd59dad1ef"

output_dir="${CEF_OUTPUT_DIR:-native/cef}"
bin_dir="${CEF_BIN_DIR:-bin}"

allow_major=()
if [[ "${CEF_ALLOW_MAJOR:-0}" == "1" ]]; then
  echo "[CEF] WARNING: allowing same-major CEF fallback for architecture smoke only."
  allow_major=(--allow-major)
fi

echo "[CEF] Resolving Electron-Go CEF target from native/CEF_VERSION..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  go run ./tools/cefresolve --json "${allow_major[@]}"
else
  go run ./tools/cefresolve --json
fi

echo "[CEF] Fetching, verifying, extracting, and staging runtime layout..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  go run ./tools/ceffetch --json --output "${output_dir}" --stage-bin "${bin_dir}" --keep-archive "${allow_major[@]}"
else
  go run ./tools/ceffetch --json --output "${output_dir}" --stage-bin "${bin_dir}" --keep-archive
fi

echo "[CEF] Verifying SHA256 for ${CEF_TARBALL}..."
if sha256sum --help 2>&1 | grep -q -- "--check"; then
  echo "${CEF_SHA256}  ${output_dir}/${CEF_TARBALL}" | sha256sum --check
else
  echo "${CEF_SHA256}  ${output_dir}/${CEF_TARBALL}" | shasum -a 256 -c
fi

echo "[CEF] Validating staged runtime layout..."
test -f "${bin_dir}/libcef.so"
go run ./tools/memaudit --root . --check-cef-layout "${bin_dir}"

rm -f "${output_dir}/${CEF_TARBALL}"

echo "[CEF] Environment staged in ${bin_dir}/"
