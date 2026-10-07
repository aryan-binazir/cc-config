---
name: pr-comments
description: Pull the current branch's active PR comments and review threads into a stable numbered checklist, then triage them discussion-first. Use when inspecting PR comments or unresolved review feedback.
---

# PR Comments

`<skill-dir>/scripts/pr_comments.py` fetches, merges, and numbers; you triage.

```bash
PRC="<skill-dir>/scripts/pr_comments.py"
uv run --script "$PRC"                                        # fetch + print checklist (default)
uv run --script "$PRC" show <n>                               # one item in full
uv run --script "$PRC" resolve <n> accepted|rejected|deferred [note...]
uv run --script "$PRC" --json                                 # full state
uv run --script "$PRC" --pr <number> ...                      # explicit PR
```

Numbers (`1`, `2` top-level; `1.1` replies) and triage decisions persist across runs; outdated threads stay, tagged, and an edited item reopens. Failures exit 1 with `{"ok": false, "error", "hint"}` — relay both to the user. `--help` covers internals and config.

## Workflow

1. Run the script and show its output verbatim (it ends with `Pick a number to discuss.`).
2. Discuss first: `show <n>` for the full text, agree on accept / reject / defer, then `resolve <n> <decision> <note>`. Code changes follow the decision.
3. Stay scoped to PR comments; capture reusable lessons in `_scratch/_agent_notes/<topic>.md`.
