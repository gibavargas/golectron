#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

manifest="${CEF_MANIFEST:-native/CEF_VERSION}"
output_dir="${CEF_OUTPUT_DIR:-native/cef}"
bin_dir="${CEF_BIN_DIR:-bin}"

allow_major=()
if [[ "${CEF_ALLOW_MAJOR:-0}" == "1" ]]; then
  echo "[CEF] WARNING: allowing same-major CEF fallback for architecture smoke only."
  allow_major=(--allow-major)
fi

echo "[CEF] Resolving Electron-Go CEF target from ${manifest}..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  go run ./tools/cefresolve --json --manifest "${manifest}" "${allow_major[@]}"
else
  go run ./tools/cefresolve --json --manifest "${manifest}"
fi

echo "[CEF] Fetching, verifying, extracting, and staging runtime layout..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  go run ./tools/ceffetch --json --manifest "${manifest}" --output "${output_dir}" --stage-bin "${bin_dir}" "${allow_major[@]}"
else
  go run ./tools/ceffetch --json --manifest "${manifest}" --output "${output_dir}" --stage-bin "${bin_dir}"
fi

echo "[CEF] Validating staged runtime layout..."
test -f "${bin_dir}/libcef.so"
go run ./tools/memaudit --root . --check-cef-layout "${bin_dir}"

echo "[CEF] Environment staged in ${bin_dir}/"
