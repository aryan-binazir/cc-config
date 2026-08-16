---
name: pr-comments
description: Pull active PR comments for the current branch into a stable numbered checklist (`_scratch/pr_reviews`) and run discussion-first triage. Use when inspecting PR comments, review threads, unresolved feedback, or a rolling PR comment checklist.
---

# PR Comments

`<skill-dir>/scripts/pr_comments.py` does the fetching, merging, and numbering; you do the triage.

## Commands

```bash
PRC="<skill-dir>/scripts/pr_comments.py"
uv run --script "$PRC"                                        # fetch + print checklist (default)
uv run --script "$PRC" show <n>                               # one item in full
uv run --script "$PRC" resolve <n> accepted|rejected|deferred [note...]
uv run --script "$PRC" --json                                 # full state
uv run --script "$PRC" --pr <number> ...                      # explicit PR
```

One GraphQL call pulls issue comments, review summaries, and review threads with thread-level `isResolved`. Active items (unresolved, unminimized; outdated threads stay, tagged) get stable numbers — `1`, `2` top-level, `1.1` replies — persisted in `<state_dir>/pr-<number>.json`; numbering, triage, and `pr-address-comments`' `agent` fields survive re-runs. An item whose body or `updatedAt` changes reopens; items deleted upstream go inactive; the script's own replies stay unnumbered. State files older than `sweep_days` are removed on fetch. Failures exit 1 with `{"ok": false, "error", "hint"}` — relay both to the user.

Config: `pr-comments.example.yaml` + gitignored `pr-comments.local.yaml` (local wins) — `provider` (github; covers github.com and GitHub Enterprise), `host` (forwarded as `GH_HOST`), `state_dir`, `sweep_days`, `agent`.

## Workflow

1. Run the script and show its output verbatim (checklist ends with `Pick a number to discuss.`).
2. Discussion first: `show <n>` for the full text, agree on accept / reject / defer, then `resolve <n> <decision> <note>`. Code changes start after the decision.
3. Stay scoped to PR comments; reusable lessons go to `_scratch/_agent_notes/<topic>.md`.
