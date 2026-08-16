---
name: pr-comments
description: Pull active PR comments for the current branch into a stable numbered checklist (`_scratch/_pr_reviews`) and run discussion-first triage. Use when inspecting PR comments, review threads, unresolved feedback, or a rolling PR comment checklist.
---

# PR Comments

`<skill-dir>/scripts/pr_comments.py` does the fetching, merging, and numbering; you do the triage.

## Commands

```bash
uv run --script "<skill-dir>/scripts/pr_comments.py"                                  # fetch + print checklist (default)
uv run --script "<skill-dir>/scripts/pr_comments.py" resolve <n> accepted|rejected|deferred [note...]
uv run --script "<skill-dir>/scripts/pr_comments.py" --json                           # full state
uv run --script "<skill-dir>/scripts/pr_comments.py" --pr <number> ...                # explicit PR
```

One GraphQL call pulls issue comments, review summaries, and review threads with thread-level `isResolved`. Active items (unresolved, current, unminimized) get stable numbers — `1`, `2` top-level, `1.1` replies — persisted in `_scratch/_pr_reviews/pr-<number>.json`; numbering, triage, and `pr-address-comments`' `agent` fields survive re-runs. An item whose body or `updatedAt` changes reopens.

## Workflow

1. Run the script and show its output verbatim (checklist ends with `Pick a number to discuss.`).
2. Discussion first: for each number picked, agree on accept / reject / defer, then `resolve <n> <decision> <note>`. Code changes start after the decision.
3. Stay scoped to PR comments; reusable lessons go to `_scratch/_agent_notes/<topic>.md`.
