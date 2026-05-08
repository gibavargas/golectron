#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

output_dir="${CEF_OUTPUT_DIR:-native/cef}"
bin_dir="${CEF_BIN_DIR:-bin}"

allow_major=()
if [[ "${CEF_ALLOW_MAJOR:-0}" == "1" ]]; then
  echo "[CEF] WARNING: allowing same-major CEF fallback for architecture smoke only."
  allow_major=(--allow-major)
fi

echo "[CEF] Resolving Electron-Go CEF target from native/CEF_VERSION..."
go run ./tools/cefresolve --json "${allow_major[@]}"

echo "[CEF] Fetching, verifying, extracting, and staging runtime layout..."
go run ./tools/ceffetch --json --output "${output_dir}" --stage-bin "${bin_dir}" "${allow_major[@]}"

echo "[CEF] Validating staged runtime layout..."
go run ./tools/memaudit --root . --check-cef-layout "${bin_dir}"

echo "[CEF] Environment staged in ${bin_dir}/"
