---
name: jira_ticket_summary
description: Generate a tight PR/JIRA summary with title, why, and what changed
version: "1.2"
---

# Generate JIRA Ticket / PR Summary

Analyze the current branch against its base branch and produce a short, paste-ready summary.

## Process

1. **Detect branch context**
   - Current branch: `git branch --show-current`
   - Base branch: `git symbolic-ref refs/remotes/origin/HEAD | cut -d'/' -f4`
   - Fallbacks: `main`, then `master`, then `develop`
   - Verify base exists: `git rev-parse --verify origin/<base-branch>`

2. **Collect diffs and commits**
   - Commits: `git log --oneline <base-branch>..HEAD`
   - Files changed: `git diff --name-status <base-branch>...HEAD`
   - Stats: `git diff --stat <base-branch>...HEAD`

3. **Summarize with minimal detail**
   - Prioritize user/business impact over implementation details
   - Group related changes into 2–5 bullets max
   - Mention risk/deployment note only if truly needed

## Output Template

```markdown
## Title
[Short PR-style title in sentence case]

## Why this change was needed
[1-2 sentences max. Problem or goal this change addresses.]

## What changed
- [Tight bullet describing the primary change]
- [Tight bullet describing secondary change]
- [Optional bullet for tests/docs/risk only if relevant]
```

## Rules

- Keep it tight; avoid fluff, repetition, and generic language.
- Do not include raw commit logs.
- Do not include sections beyond **Title**, **Why this change was needed**, and **What changed**.
- If information is missing, state assumptions briefly and continue.

Usage: Run from any branch to generate a concise summary for JIRA or PR description.
