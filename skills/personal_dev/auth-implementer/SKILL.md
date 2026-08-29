---
name: auth-implementer
description: Delegate a well-defined change to a worker that gets it done in one hour and returns a reviewed PR. Use when the user invokes /auth-implementer.
disable-model-invocation: true
---

# Auth Implementer

One worker, one hour, one reviewed PR, zero questions back. Mechanics live in the `implementer` skill.

Write `_scratch/implementer/<slug>.md` — **Goal**, **Evidence** (file:line), **Scope** (owned files; concurrent branches nearby), **Shape** (design it), **Allowed fixes** (or `None`), **Acceptance** (checkable) — then append [`AUTONOMY.md`](AUTONOMY.md).

```bash
uv run ~/repos/cc-config/skills/personal_dev/lead/scripts/delegate.py \
  --worker <name> --worktree --timeout-ms 3600000 --prompt-file <absolute path>
```

On `timed_out`, resume in the same worktree (`--cwd`) with the progress file's remaining steps. Rebase, gate, merge on green when authorized; judge the diff yourself.
