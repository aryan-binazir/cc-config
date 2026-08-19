---
name: nvim
description: Use only when the user explicitly invokes $nvim or /nvim to open the currently discussed code, file, test, symbol, or location in Neovim — in a new tmux window when inside tmux, otherwise by printing a paste-able tmux command targeting the repo's session.
disable-model-invocation: true
---

# Nvim

Open the current code/document reference in a new tmux window with `nvim`.

## Behavior

- `$nvim` and `/nvim` are navigation commands: resolve the location and open it.
- The text after the invocation names the target, such as `the skill`, `the failing test`, or `end of file`.
- Prefer exact evidence over inference; when the target is ambiguous, ask one short question.

## Resolution Order

1. Explicit `path:line[:column]` in the user's message or recent command output.
2. The last file and line range the agent read, quoted, reviewed, or discussed.
3. Failing test, compiler, linter, stack trace, or review output containing a file and line.
4. Symbol, function, type, route, config key, test name, or text snippet found with `rg -n`.
5. Paired files by repo convention (implementation/test, handler/spec), only when the intended target is clear.

When searching, run focused commands such as:

```bash
rg -n --hidden -g '!vendor' -g '!node_modules' -g '!.git' 'SymbolOrSnippet'
```

With multiple plausible matches, show the short list and ask which one.

## Open Command

Always use the bundled helper:

```bash
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
~/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" path/to/file.go 150
```

With a column:

```bash
~/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" path/to/file.go 150 14
```

The helper has two modes:

- Inside tmux (`$TMUX` set): opens a new window in the current session and runs nvim in its shell (the window survives quitting nvim); prints nothing.
- Outside tmux (GUI apps such as t3code, plain shells): the agent cannot know which terminal the user is looking at, so the helper prints a paste-able `tmux new-window ... \; send-keys ... \; switch-client` command targeting the tmux session named after the repo (worktrees resolve to their parent repo). If no session matches it prints one line per existing session; if there are no sessions it prints a plain `cd ... && nvim ...` line.

## Reporting

Inside tmux, respond with only the opened location:

```text
Opened internal/api/server/routes.go at line 84.
```

Outside tmux, relay the helper's output verbatim in a bash code block so the user can copy it, e.g.:

```bash
# opens in tmux session cc-config
tmux new-window -t cc-config: -c /home/ar/repos/cc-config \; send-keys "nvim +84 -- internal/api/server/routes.go" Enter \; switch-client -t cc-config
```

If the helper listed several sessions, show the list and ask which one. Add explanation only when the user asks.
