---
description: Brutally honest review of uncommitted changes
version: "2.2"
---

# Uncommitted Review

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

## Review Focus

1. **Correctness**: Does this code actually work? Logic errors, broken algorithms, wrong assumptions.

2. **Regressions**: Will committing this break existing functionality? Removed behavior, changed contracts, broken integrations.

3. **Security**: Injection risks, auth issues, data exposure, secrets in code.

4. **Performance**: N+1 queries, unnecessary loops, memory leaks, expensive operations.

5. **Maintainability**: Naming, complexity, duplication, missing error handling, test coverage gaps.

6. **Edge Cases**: What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

**Important**: Only use information from the diff. If you're unsure whether something is an issue, say so rather than guessing.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Summary
[2-4 sentences: what these changes do, what motivated them, and what
areas of the codebase are affected. Plain language for someone
unfamiliar with these changes.]

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
