---
name: nvim
description: Use only when the user explicitly invokes $nvim or /nvim to open the currently discussed code, file, test, symbol, or location in a new tmux window with Neovim.
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

Prefer the bundled helper:

```bash
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
/home/ar/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" path/to/file.go 150
```

With a column:

```bash
/home/ar/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" path/to/file.go 150 14
```

Fallback if the helper is unavailable:

```bash
tmux new-window -c "$repo_root" "nvim +150 -- 'path/to/file.go'"
```

## Reporting

Respond with only the opened location; add explanation only when the user asks.

```text
Opened internal/api/server/routes.go at line 84.
```
