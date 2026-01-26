---
description: Brutally honest review of committed code since branch diverged
version: "2.0"
---

# Code Review

Review all committed code on this branch since it diverged from base. Be brutally honest.

## Get Changes

```bash
# Auto-detect base branch
git diff origin/main..HEAD      # Try main first
git diff origin/master..HEAD    # Fallback to master

# Context
git log --oneline origin/main..HEAD
git diff --stat origin/main..HEAD
```

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
