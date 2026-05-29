#!/usr/bin/env bash
set -euo pipefail

usage() {
  printf 'usage: %s <repo-root> <file> [line] [column]\n' "${0##*/}" >&2
}

if [ "$#" -lt 2 ] || [ "$#" -gt 4 ]; then
  usage
  exit 2
fi

repo_root=$1
file=$2
line=${3:-}
column=${4:-}

if [ ! -d "$repo_root" ]; then
  printf 'repo root does not exist: %s\n' "$repo_root" >&2
  exit 1
fi

case "$file" in
  /*) file_path=$file ;;
  *) file_path=$repo_root/$file ;;
esac

if [ ! -e "$file_path" ]; then
  printf 'file does not exist: %s\n' "$file" >&2
  exit 1
fi

if [ -n "$line" ] && ! [[ "$line" =~ ^[0-9]+$ ]]; then
  printf 'line must be numeric: %s\n' "$line" >&2
  exit 2
fi

if [ -n "$column" ] && ! [[ "$column" =~ ^[0-9]+$ ]]; then
  printf 'column must be numeric: %s\n' "$column" >&2
  exit 2
fi

if [ -n "$line" ] && [ -n "$column" ]; then
  nvim_cmd=(nvim "+call cursor($line, $column)" "+normal! zz" -- "$file")
elif [ -n "$line" ]; then
  nvim_cmd=(nvim "+$line" -- "$file")
else
  nvim_cmd=(nvim -- "$file")
fi

printf -v command '%q ' "${nvim_cmd[@]}"

if [ -n "${TMUX:-}" ]; then
  tmux new-window -c "$repo_root" "$command"
  exit 0
fi

attached_session=$(
  { tmux list-sessions -F '#{session_attached} #{session_name}' 2>/dev/null || true; } |
    awk '$1 > 0 {print $2; exit}'
)

if [ -n "$attached_session" ]; then
  tmux new-window -t "${attached_session}:" -c "$repo_root" "$command"
  exit 0
fi

if tmux has-session -t codex-nvim 2>/dev/null; then
  tmux new-window -t codex-nvim: -c "$repo_root" "$command"
else
  tmux new-session -d -s codex-nvim -c "$repo_root" "$command"
fi
