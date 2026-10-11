#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat >&2 <<'USAGE'
usage: open_nvim_tmux.sh [--print] <repo-root> <file> [line] [column]

Inside Herdr (HERDR_ENV=1) or tmux (TMUX set): opens the file in nvim in a
pane left of the calling agent's pane, reusing that nvim on later calls.
Outside Herdr and tmux (GUI apps, t3code, plain shells): prints a paste-able
tmux command targeting the tmux session named after the repo. --print forces
the print behavior.
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

# Inside Herdr or tmux, nvim lives in a pane left of the agent, paired to it and
# reachable over its --listen socket, so later opens reuse it.
socket_dir=${XDG_RUNTIME_DIR:-${TMPDIR:-/tmp}}/agent-nvim

jump_to_file() {
  for _ in $(seq 60); do
    nvim --server "$socket" --remote-expr 1 >/dev/null 2>&1 && break
    sleep 0.05
  done
  local jump="edit ' .. fnameescape('${file_path//\'/\'\'}')"
  [ -n "$line" ] && jump+=" .. ' | call cursor($line, ${column:-1}) | normal! zz'"
  nvim --server "$socket" --remote-send '<C-\><C-N>'
  nvim --server "$socket" --remote-expr "execute('$jump)" >/dev/null
}

if [ "$print_only" -eq 0 ] && [ "${HERDR_ENV:-}" = 1 ]; then
  agent=${HERDR_PANE_ID:?HERDR_PANE_ID is required inside Herdr}
  key=${agent//[^[:alnum:]_-]/-}
  socket=$socket_dir/herdr-$key.sock
  label=agent-nvim-$key
  mkdir -p -m 700 "$socket_dir"
  editor=$(herdr pane list --workspace "${HERDR_WORKSPACE_ID:?HERDR_WORKSPACE_ID is required inside Herdr}" |
    jq -r --arg label "$label" '[.result.panes[] | select(.label == $label)][0].pane_id // empty')

  if [ -n "$editor" ]; then
    jump_to_file
  else
    rm -f "$socket"
    # Herdr splits only right or down; the 65% left child becomes nvim's after the swap.
    editor=$(herdr pane split --pane "$agent" --direction right --ratio 0.65 --cwd "$repo_root" | jq -er '.result.pane.pane_id')
    herdr pane swap --source-pane "$agent" --target-pane "$editor" >/dev/null
    herdr pane rename "$editor" "$label" >/dev/null
    printf -v listen_command '%q ' nvim --listen "$socket" "${nvim_cmd[@]:1}"
    herdr pane run "$editor" "exec ${listen_command% }" >/dev/null
  fi
  herdr pane focus --pane "$agent" --direction left >/dev/null

  # Label the agent the way nvim's <leader>pi/cc/cd do, so nvim's sends reach it.
  agent_name=$(herdr pane get "$agent" | jq -r '.result.pane.agent // empty')
  case "$agent_name" in
    claude | codex | opencode | pi) herdr pane rename "$agent" "nvim-ai-${editor//[^[:alnum:]_-]/-}-$agent_name" >/dev/null ;;
  esac
  exit 0
fi

if [ "$print_only" -eq 0 ] && [ -n "${TMUX:-}" ] && [ -n "${TMUX_PANE:-}" ]; then
  agent=$TMUX_PANE
  socket=$socket_dir/tmux-$(tmux display-message -p '#{pid}')-${agent#%}.sock
  mkdir -p -m 700 "$socket_dir"
  editor=$(tmux list-panes -t "$agent" -F '#{pane_id} #{@agent_nvim_for}' | awk -v a="$agent" '$2 == a { print $1; exit }')

  if [ -n "$editor" ]; then
    jump_to_file
    tmux select-pane -t "$editor"
  else
    rm -f "$socket"
    editor=$(tmux split-window -h -b -l 65% -t "$agent" -c "$repo_root" -P -F '#{pane_id}' \
      nvim --listen "$socket" "${nvim_cmd[@]:1}")
    tmux set-option -p -t "$editor" @agent_nvim_for "$agent" \; set-option -p -t "$editor" @agent_nvim_socket "$socket"
  fi

  # Mark the agent the way nvim's <leader>pi/cc/cd do, so nvim's sends reach it.
  agent_name=$(tmux display-message -p -t "$agent" '#{pane_current_command}')
  case "$agent_name" in
    claude | codex | opencode | pi) tmux set-option -p -t "$agent" @nvim_ai_pane "$agent_name" ;;
  esac
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
