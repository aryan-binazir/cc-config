---
name: context-sync
description: Post the final task status as a comment on the relevant Jira or Linear issue. Use when finishing, pausing, blocking, or handing off ticket-backed work and the external tracker needs the final status synced. Detect Jira vs Linear from explicit issue links, the source ticket already used in the conversation, branch/PR/commit references resolved through available tools, and repo environment. Local `_scratch/_context` files stay untouched.
---

# Context Sync

Post a concise final status comment to the external ticket that owns the work.

## Scope

The whole deliverable is exactly one final status comment in Jira or Linear. Everything else — ticket descriptions, labels, fields, assignees, priorities, workflow status, `_scratch/_context` files, PRs, commits, and code — stays untouched, with one exception: edit ticket metadata when the user explicitly asks for it.

Claim only what the conversation or verified repo state supports. Stop and ask for the issue URL or key when the target can't be determined confidently.

## Target Detection

Use this order:

1. An explicit issue URL or key from the user's current request.
2. The source ticket already fetched or discussed in the conversation.
3. Ticket references from the current branch, recent commits, PR title/body, and local repo rules.
4. Resolve the candidate issue through available Jira or Linear tooling.
5. A URL host that clearly identifies the tracker decides it: `linear.app` or known Linear workspace URLs → Linear; Atlassian/Jira hosts → Jira.
6. A bare issue key such as `ABC-123` fits both trackers; resolve it with available tools.
7. If both Jira and Linear resolve, or neither does, ask the user which issue to comment on.

Environment signals are supporting evidence, not proof: a branch name or repo convention can identify a candidate, but verify the issue exists in the chosen tracker before posting.

## Status Collection

Build the comment from factual state only:

- Current outcome: `Complete`, `Blocked`, or `Partial`.
- What changed or was done.
- What was verified, with exact commands when known, and what was left unverified if relevant.
- PR, branch, commit, or artifact links when available.
- Remaining work, blockers, or follow-up owners.

Use the conversation first; check repo state when it keeps the status honest:

```bash
git status -sb
git log --oneline -5
```

Include the PR link when one exists and GitHub tooling is available. Run expensive checks only when the user asked for fresh verification.

## Comment Format

Keep it short and scannable; omit empty sections. For blocked or partial work, make the blocker obvious in the first two lines.

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

## Posting Rules

- Prefer installed MCP/app tools for Jira or Linear; use CLI or API tooling only when it is already configured.
- Work with existing tooling and credentials only, and post only through tracker tools or APIs.
- If a write-capable tracker tool is unavailable, report the blocker and include the exact comment body that should be posted.
- After posting, reply with the issue key or URL, tracker name, and a one-sentence summary of what was posted.
