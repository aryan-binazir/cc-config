---
description: Parallel review of uncommitted changes
version: "1.1"
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

- **Agent 1: Correctness & Regressions** - Does this code actually work? Logic errors, broken algorithms, wrong assumptions. Will committing break existing functionality? Removed behavior, changed contracts.
- **Agent 2: Security & Performance** - Injection risks, auth issues, data exposure, secrets in code. N+1 queries, unnecessary loops, memory leaks, expensive operations.
- **Agent 3: Maintainability & Edge Cases** - Naming, complexity, duplication, missing error handling, test coverage gaps. What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

Be specific. Point out exactly what's wrong and where. No padding.

**Important**: Only use information from the diff. If you're unsure whether something is an issue, say so rather than guessing.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Critical (must fix before commit)
- [file:line] - [what's wrong and why it matters]

## High (should fix)
- [file:line] - [what's wrong and why it matters]

## Low (consider fixing)
- [file:line] - [what's wrong and why it matters]

## Uncertain
- [file:line] - [potential issue, but unsure - explain why]

## Verdict
[GOOD TO COMMIT / NEEDS FIXES] - [1 sentence summary]
```

If no issues found, say so and move on.
