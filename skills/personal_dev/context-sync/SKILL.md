---
name: context-sync
description: Post the final task status as a comment on the relevant Jira or Linear issue. Use when finishing, pausing, blocking, or handing off ticket-backed work and the external tracker needs the final status synced. Detect Jira vs Linear from explicit issue links, the source ticket already used in the conversation, branch/PR/commit references resolved through available tools, and repo environment. Do not maintain local `_scratch/_context` files.
---

# Context Sync

Post a concise final status comment to the external ticket that owns the work.

This skill is deliberately narrow. It is not a project-notes workflow anymore.

## Scope

- Post exactly one final status update as an issue comment in Jira or Linear.
- Do not create or update `_scratch/_context` files.
- Do not edit ticket descriptions, labels, fields, assignees, priorities, or workflow status unless the user explicitly asks.
- Do not open PRs, commit code, or make implementation changes.
- Do not claim tests passed, code shipped, or review completed unless that is supported by the conversation or verified repo state.
- If the target tracker or issue cannot be determined confidently, stop and ask for the issue URL or key.

## Target Detection

Use this order:

1. Prefer an explicit issue URL or key from the user's current request.
2. Prefer the source ticket already fetched or discussed in the current conversation.
3. Inspect the current branch, recent commits, PR title/body, and local repo rules for ticket references.
4. Resolve the candidate issue through available Jira or Linear tooling.
5. If a URL host clearly identifies the tracker, use that tracker:
   - `linear.app` or known Linear workspace URLs -> Linear
   - Atlassian/Jira hosts -> Jira
6. If only an issue key such as `ABC-123` is available, do not assume Jira or Linear from the format alone. Resolve it with available tools.
7. If both Jira and Linear resolve, or neither resolves, ask the user which issue to comment on.

Environment signals are supporting evidence, not proof. A branch name, ticket-shaped key, or repo convention can identify a candidate issue, but the skill must still verify that the issue exists in the chosen tracker before posting.

## Status Collection

Build the comment from factual state only:

- Current outcome: `Complete`, `Blocked`, or `Partial`.
- What changed or was done.
- What was verified, including exact commands when known.
- What was not verified, if relevant.
- PR, branch, commit, or artifact links when available.
- Remaining work, blockers, or follow-up owners.

Use the conversation first. Check repo state when it helps avoid stale or invented status:

```bash
git status -sb
git log --oneline -5
```

If a PR exists and GitHub tooling is available, include the PR link. Do not run expensive checks just to produce a status comment unless the user asked for fresh verification.

## Comment Format

Keep the comment short and scannable:

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

Omit empty sections. For blocked or partial work, make the blocker obvious in the first two lines.

## Posting Rules

- Prefer installed MCP/app tools for Jira or Linear when available.
- Use CLI or API tooling only when it is already configured in the environment.
- Do not install new tools or create new credentials.
- Do not use browser automation for tracker writes.
- If a write-capable tracker tool is unavailable, report the blocker and include the exact comment body that should be posted.
- After posting, reply with the issue key or URL, tracker name, and a one-sentence summary of what was posted.
