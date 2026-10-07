---
name: pr-address-comments
description: Patch, commit, and reply to my `agent:`-prefixed comments on the current PR.
disable-model-invocation: true
---

# PR Address Comments

Turn the authenticated user's `agent:` PR comments into a local commit and reply with its hash. The sibling `pr-comments` skill's script fetches and replies, sharing that skill's state and config:

```bash
PRC="<pr-comments-skill-dir>/scripts/pr_comments.py"
uv run --script "$PRC" --json                          # fetch + full state
uv run --script "$PRC" show <n>                        # full text of one item
uv run --script "$PRC" reply <n> --commit <hash> [--agent <label>] [--testing "<cmd>"] [--body "<text>"]
```

## Workflow

1. `--json` (add `--pr <number>` when the branch has no PR). Actionable items: `author` exactly matches `gh api user --jq .login`, `status` is `open`, and the first non-empty, non-quoted body line starts with `agent:` (case-insensitive). The instruction is that line's remainder plus the rest of the body. Everything else — other authors, the user's unprefixed comments, parent threads, paths, hunks — is context.
2. Implement every open actionable comment that is safe to handle together; run focused checks for the patch.
3. Commit only the files changed for these instructions, per repo commit rules (fallback: `<type>(<scope>): Address PR agent comments`, reusing the branch's existing commit type and scope).
4. Push, then `reply <n> --commit <hash> --agent <your name>` (Claude, Codex, Cursor…; config `agent` is the fallback) per handled comment; every reply carries the hash, alone or in a sentence. The script picks the GitHub target and records the reply. Failures exit 1 with `{"ok": false, "error", "hint"}` — relay both to the user.

## Rules

- Resolve GitHub threads only on the user's explicit request.
- Commit real code changes only. When an existing commit already satisfies an instruction, reply pointing at it; otherwise ask.
- Stop and ask when instructions conflict, are ambiguous, or would widen scope beyond the comment.
- Leave unrelated worktree changes untouched; where they overlap files you edit, preserve the user's work.

## Output

1. PR title and URL.
2. Each handled `agent:` comment with source URL and commit hash.
3. Checks run, or why they were skipped.
4. Comments left open and why.
