#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
usage: open_nvim_tmux.sh [--print] <repo-root> <file> [line] [column]

Inside tmux (TMUX set): opens the file in a new window of the current session.
Outside tmux (GUI apps, t3code, plain shells): prints a paste-able tmux command
instead, targeting the tmux session named after the repo. --print forces the
print behavior.
USAGE
}

print_only=0
if [ "${1:-}" = "--print" ]; then
  print_only=1
  shift
fi

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
repo_root=$(cd "$repo_root" && pwd)

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
command=${command% }

if [ "$print_only" -eq 0 ] && [ -n "${TMUX:-}" ]; then
  # Open a shell first and type the command into it, so the window survives
  # quitting nvim.
  tmux new-window -c "$repo_root" \; send-keys "$command" Enter
  exit 0
fi

# Not inside tmux: print a command to paste. Target the tmux session named
# after the repo (worktrees resolve to their parent repo), so the window lands
# in the session where the user works on this codebase; switch-client jumps
# there when pasted inside any tmux pane. The window opens a shell and types
# the nvim command so it survives quitting nvim.
common_dir=$(git -C "$repo_root" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)
if [ -n "$common_dir" ]; then
  repo_name=$(basename "$(dirname "$common_dir")")
else
  repo_name=$(basename "$repo_root")
fi

sessions=$(tmux list-sessions -F '#{session_name}' 2>/dev/null || true)

if grep -qxF "$repo_name" <<<"$sessions"; then
  printf '# opens in tmux session %s\n' "$repo_name"
  printf 'tmux new-window -t %q: -c %q \\; send-keys "%s" Enter \\; switch-client -t %q\n' "$repo_name" "$repo_root" "$command" "$repo_name"
  exit 0
fi

if [ -n "$sessions" ]; then
  printf '# no tmux session named %s; pick one:\n' "$repo_name"
  while read -r session; do
    printf 'tmux new-window -t %q: -c %q \\; send-keys "%s" Enter \\; switch-client -t %q\n' "$session" "$repo_root" "$command" "$session"
  done <<<"$sessions"
  exit 0
fi

printf '# no tmux sessions; run in any shell:\n'
printf 'cd %q && %s\n' "$repo_root" "$command"
