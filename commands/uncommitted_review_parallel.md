---
description: Parallel review of uncommitted changes
version: "1.0"
---

# Uncommitted Review (Parallel)

Review ONLY staged and unstaged changes. Nothing else.

## Scope

**ONLY review**:
- Staged changes (git diff --cached)
- Unstaged changes (git diff)

**DO NOT review**:
- Already committed code
- Files not modified in the working tree

## Get Changes

```bash
# Staged changes
git diff --cached

# Unstaged changes
git diff

# Summary of all uncommitted changes
git status --short
```

## Parallel Review

Spawn sub-agents to check in parallel:

- **Agent 1: Correctness** - Does this code actually work? Logic errors, broken algorithms, wrong assumptions, failed edge cases.
- **Agent 2: Regressions** - Will committing this break existing functionality? Removed behavior, changed contracts, broken integrations.
- **Agent 3: Concerns** - Security issues, performance problems, maintainability red flags, missing error handling, test coverage gaps.

Be specific. Point out exactly what's wrong and where. No padding.

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
[GOOD TO COMMIT / NEEDS FIXES] - [1 sentence summary]
```

If no issues found, say so and move on.
