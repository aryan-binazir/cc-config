---
name: nvim
description: Open the discussed code, file, test, symbol, or location in Neovim. Use only when the user explicitly invokes $nvim or /nvim.
disable-model-invocation: false
---

# Nvim

`$nvim` and `/nvim` are navigation commands: resolve the location and open it. Any text after the invocation names the target (`the skill`, `the failing test`, `end of file`). Prefer exact evidence over inference; when the target is ambiguous, ask one short question, listing the plausible matches.

## Resolution Order

1. Explicit `path:line[:column]` in the user's message or recent command output.
2. The last file and line range the agent read, quoted, reviewed, or discussed.
3. Failing test, compiler, linter, stack trace, or review output containing a file and line.
4. Symbol, function, type, route, config key, test name, or text snippet found with `rg -n --hidden -g '!vendor' -g '!node_modules' -g '!.git'`.
5. Paired files by repo convention (implementation/test, handler/spec), only when the intended target is clear.

## Open

Always use the bundled helper (line and column optional):

```bash
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
~/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" path/to/file.go 150 14
```

Inside Herdr or tmux, it opens a new tab or window and prints nothing; reply with only the location:

```text
Opened internal/api/server/routes.go at line 84.
```

Elsewhere (GUI apps such as t3code, plain shells) you cannot see the user's terminal, so it prints a paste-able command targeting the tmux session named after the repo: relay its output verbatim in a bash code block. When no session matches, it prints one option per session; show them and ask which. Add explanation only when the user asks.
