#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd -- "${BASH_SOURCE[0]%/*}" && pwd)"
uv_path="$(command -v uv)"
command=("$uv_path" run --script "$script_dir/call.py" "$@")

if [[ -v T3_MCP_BEARER_TOKEN ]]; then
  systemd_run="$(command -v systemd-run)"
  exec "$systemd_run" \
    --user \
    --pipe \
    --wait \
    --collect \
    --quiet \
    --service-type=exec \
    "--working-directory=$PWD" \
    "${command[@]}"
fi

exec "${command[@]}"
