---
name: hand-code
description: Find one small, worthwhile code change for me to hand-code in the current repo, then open it in Neovim.
disable-model-invocation: true
---

# Hand Code

Scout only. Read and report. Do not edit, commit, or create branches or tickets.

Require one mode keyword. If neither is present, ask.

- `learn`: The language is new to me. Pick a contained task with nearby tests or examples to copy from. Name the language features I will practice, but do not solve the task.
- `sharp`: I know the language. Reading and tracing the existing code is part of the exercise, so give minimal clues.

If I name a ticket or feature branch, pick unfinished work within its existing scope. Otherwise pick from anywhere in the repo.

Find one real change I can finish in one sitting: a bug you can show, missing behavior, or concrete maintenance pain. Coverage numbers, style preferences, and imagined refactors do not count. If there is no honest task, say so.

Tell me what to change, why it matters with `file:line` evidence, what done means, and the command that proves it. Then open the starting file and line:

```bash
repo_root=$(git rev-parse --show-toplevel 2>/dev/null || pwd)
~/repos/cc-config/skills/personal_dev/nvim/scripts/open_nvim_tmux.sh "$repo_root" <file> <line>
```
