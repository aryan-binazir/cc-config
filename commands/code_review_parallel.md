---
description: Parallel review of committed code since branch diverged
version: "1.2"
---

# Code Review (Parallel)

Review ONLY the changes made on the current branch compared to main. Nothing else.

## Scope

**ONLY review**:
- Code changes introduced on this branch
- Commits between where this branch diverged from main and HEAD

**DO NOT review**:
- Files not modified by this branch
- Changes from rebases, merges, or upstream commits
- Code that existed before this branch was created

## Get Changes

```bash
# Find the merge-base (where this branch diverged from main)
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)

# Get ONLY the diff between merge-base and current HEAD
git diff $BASE..HEAD

# List commits on this branch only (exclude merge commits)
git log --oneline --no-merges $BASE..HEAD

# Summary of files changed on this branch
git diff --stat $BASE..HEAD
```

**Important**: If a file appears in the diff that wasn't intentionally modified on this branch, ignore it - it's likely a rebase artifact.

## Parallel Review

Spawn sub-agents to check in parallel:

- **Agent 1: Correctness & Regressions** - Does this code actually work? Logic errors, broken algorithms, wrong assumptions. Will merging break existing functionality? Removed behavior, changed contracts.
- **Agent 2: Security & Performance** - Injection risks, auth issues, data exposure, secrets in code. N+1 queries, unnecessary loops, memory leaks, expensive operations.
- **Agent 3: Maintainability & Edge Cases** - Naming, complexity, duplication, missing error handling, test coverage gaps. What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

Be specific. Point out exactly what's wrong and where. No padding.

**Important**: Only use information from the diff. If you're unsure whether something is an issue, say so rather than guessing.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Summary
[2-4 sentences: what these changes do, what motivated them, and what
areas of the codebase are affected. Plain language for someone
unfamiliar with this branch.]

## Critical (must fix before merge)
- [file:line] - [what's wrong and why it matters]

## High (should fix)
- [file:line] - [what's wrong and why it matters]

## Low (consider fixing)
- [file:line] - [what's wrong and why it matters]

## Uncertain
- [file:line] - [potential issue, but unsure - explain why]

## Verdict
[APPROVE / NEEDS FIXES / REJECT] - [1 sentence summary]
```

If no issues found, say so and move on.
