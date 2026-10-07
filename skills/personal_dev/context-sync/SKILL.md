---
name: context-sync
description: Post a final status comment on the Jira or Linear issue that owns the work. Use when the tracker needs the status of finished, paused, blocked, or handed-off ticket work.
---

# Context Sync

The deliverable is exactly one concise final status comment on the owning ticket. Ticket descriptions, labels, fields, assignees, priorities, workflow status, `_scratch/_context` files, PRs, commits, and code stay untouched; edit ticket metadata only on explicit request.

## Target Detection

Find the candidate, in priority order:

1. An explicit issue URL or key in the current request.
2. The source ticket already fetched or discussed in the conversation.
3. Ticket references in the current branch, recent commits, PR title/body, and local repo rules: candidates until verified.

Pick the tracker: a `linear.app` or known Linear workspace URL → Linear; an Atlassian/Jira host → Jira; a bare key like `ABC-123` fits both, so resolve it through available tools. Verify the issue exists in the chosen tracker before posting. When both trackers resolve, neither does, or the target stays uncertain, ask for the issue URL or key.

## Status Collection

Claim only what the conversation or verified repo state supports:

- Outcome: `Complete`, `Blocked`, or `Partial`.
- What changed or was done.
- What was verified, with exact commands when known, and what was left unverified if relevant.
- PR, branch, commit, or artifact links when available.
- Remaining work, blockers, or follow-up owners.

Use the conversation first; check `git status -sb` and `git log --oneline -5` when they keep the status honest. Look up the PR link with GitHub tooling when available. Reserve expensive checks for when the user asks for fresh verification.

## Comment Format

Short and scannable; omit empty sections. For blocked or partial work, make the blocker obvious in the first two lines.

```md
Final status: Complete

Summary:
- ...

Validation:
- `...` passed
- Not run: ...

Links:
- PR: ...
- Branch: ...

Remaining:
- None
```

## Posting

- Post through installed MCP/app tracker tools, else CLI or API tooling already configured with existing credentials.
- With no write-capable tool, report the blocker plus the exact comment body to post.
- After posting, reply with the issue key or URL, tracker name, and a one-sentence summary of what was posted.
