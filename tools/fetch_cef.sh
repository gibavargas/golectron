#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${repo_root}"

manifest="${CEF_MANIFEST:-native/CEF_VERSION}"
output_dir="${CEF_OUTPUT_DIR:-native/cef}"
bin_dir="${CEF_BIN_DIR:-bin}"
resolve_timeout="${CEF_RESOLVE_TIMEOUT:-30s}"
fetch_timeout="${CEF_FETCH_TIMEOUT:-30m}"

allow_major=()
if [[ "${CEF_ALLOW_MAJOR:-0}" == "1" ]]; then
  echo "[CEF] WARNING: allowing same-major CEF fallback for architecture smoke only."
  allow_major=(--allow-major)
fi
read -r -a cefresolve_cmd <<< "${CEFRESOLVE_BIN:-go run ./tools/cefresolve}"
read -r -a ceffetch_cmd <<< "${CEFFETCH_BIN:-go run ./tools/ceffetch}"
read -r -a memaudit_cmd <<< "${MEMAUDIT_BIN:-go run ./tools/memaudit}"

echo "[CEF] Resolving Electron-Go CEF target from ${manifest}..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  "${cefresolve_cmd[@]}" --json --manifest "${manifest}" --timeout "${resolve_timeout}" "${allow_major[@]}"
else
  "${cefresolve_cmd[@]}" --json --manifest "${manifest}" --timeout "${resolve_timeout}"
fi

echo "[CEF] Fetching, verifying, extracting, and staging runtime layout..."
if [[ ${#allow_major[@]} -gt 0 ]]; then
  "${ceffetch_cmd[@]}" --json --manifest "${manifest}" --output "${output_dir}" --stage-bin "${bin_dir}" --timeout "${fetch_timeout}" "${allow_major[@]}"
else
  "${ceffetch_cmd[@]}" --json --manifest "${manifest}" --output "${output_dir}" --stage-bin "${bin_dir}" --timeout "${fetch_timeout}"
fi

echo "[CEF] Validating staged runtime layout..."
test -f "${bin_dir}/libcef.so"
"${memaudit_cmd[@]}" --root . --check-cef-layout "${bin_dir}"

echo "[CEF] Environment staged in ${bin_dir}/"
