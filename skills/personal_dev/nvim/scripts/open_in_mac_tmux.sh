#!/usr/bin/env bash
# Open nvim in a tmux window of the repo's session and switch the user's
# remote (SSH-attached, e.g. laptop) tmux client to it. Runs on the machine
# that hosts the tmux server (e.g. from a t3code project action).
#
# usage: open_in_mac_tmux.sh <repo-root> [file] [line] [column]
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 4 ]; then
  printf 'usage: %s <repo-root> [file] [line] [column]\n' "${0##*/}" >&2
  exit 2
fi

repo_root=$(cd "$1" && pwd)
file=${2:-.}
line=${3:-}
column=${4:-}

if [ -n "$line" ] && [ -n "$column" ]; then
  nvim_cmd=(nvim "+call cursor($line, $column)" "+normal! zz" -- "$file")
elif [ -n "$line" ]; then
  nvim_cmd=(nvim "+$line" -- "$file")
else
  nvim_cmd=(nvim -- "$file")
fi
printf -v command '%q ' "${nvim_cmd[@]}"
command=${command% }

# Session named after the repo (worktrees resolve to the parent repo).
common_dir=$(git -C "$repo_root" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)
if [ -n "$common_dir" ]; then
  session=$(basename "$(dirname "$common_dir")")
else
  session=$(basename "$repo_root")
fi

if ! tmux has-session -t "=$session" 2>/dev/null; then
  tmux new-session -d -s "$session" -c "$repo_root"
fi

tmux new-window -t "$session:" -c "$repo_root" \; send-keys "$command" Enter

# Remote clients: attached ttys whose `who` origin is a host/IP, not (tmux...).
switched=0
while read -r tty; do
  [ -n "$tty" ] || continue
  origin=$(who 2>/dev/null | awk -v t="${tty#/dev/}" '$2==t {print $NF}')
  case "$origin" in
    \(tmux*|"") ;;
    \(*\)) tmux switch-client -c "$tty" -t "$session" && switched=1 ;;
  esac
done < <(tmux list-clients -F '#{client_tty}' 2>/dev/null || true)

if [ "$switched" -eq 1 ]; then
  printf 'opened %s in tmux session %s and switched your remote client to it\n' "$file" "$session"
else
  printf 'opened %s in tmux session %s (no remote tmux client to switch; run: tmux switch-client -t %s)\n' "$file" "$session" "$session"
fi
