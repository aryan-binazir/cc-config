---
name: jira-ticket-summary
description: Generate a tight, paste-ready JIRA or PR summary for the current branch with a title, brief rationale, and a short list of what changed. Use when the user wants a concise PR summary, ticket update, or branch summary.
---

# Jira Ticket Summary

Analyze the current branch against its base branch and produce a short summary focused on user or business impact.

## Workflow

1. Determine the current branch.
2. Detect the most likely base branch from `origin/HEAD`, falling back to `main`, `master`, then `develop`.
3. Verify that the base ref exists.
4. Gather commits, changed files, and diff stats between the base branch and `HEAD`.
5. Summarize with minimal implementation detail.

## Output Template

```markdown
## Title
[Short PR-style title in sentence case]

## Why this change was needed
[1 to 2 sentences]

## What changed
- [Primary change]
- [Secondary change]
- [Optional tests, docs, or risk note if truly relevant]
```

## Rules

- Keep it tight.
- Prioritize impact over implementation mechanics.
- Do not include raw commit logs.
- Do not add extra sections.
- If something is missing, state the assumption briefly and continue.
