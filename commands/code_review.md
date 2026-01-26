---
description: Brutally honest review of committed code since branch diverged
version: "2.1"
---

# Code Review

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
# Get current branch name
BRANCH=$(git branch --show-current)

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

## Review Focus

1. **Correctness**: Does this code actually work? Logic errors, broken algorithms, wrong assumptions, failed edge cases.

2. **Regressions**: Will merging this break existing functionality? Removed behavior, changed contracts, broken integrations.

3. **Concerns**: Security issues, performance problems, maintainability red flags, missing error handling.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Issues

1. [file:line] - [what's wrong and why it matters]
2. [file:line] - [what's wrong and why it matters]
...

## Concerns

- [Any security, performance, or architectural concerns]

## Verdict
[APPROVE / NEEDS FIXES / REJECT] - [1 sentence summary]
```

If no issues found, say so and move on.
