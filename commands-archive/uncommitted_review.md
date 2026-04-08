---
name: uncommitted_review
description: Brutally honest review of uncommitted changes
version: "3.0"
---

# Uncommitted Review

Be brutally honest. Review ONLY staged and unstaged changes. Nothing else.

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

## PR Context

If a PR exists for this branch, read the full PR (description, diff, and comments) first — it contains decisions and context that inform the review.

## Understand Context

Before flagging issues, **read the surrounding code** in each changed file. The diff alone is not enough — you need context to judge correctness, types, contracts, and intent. Open the full file around changed lines.

## Review Focus

Only flag real issues. Do not stretch to fill categories — if a category has no issues, skip it.

1. **Correctness**: Logic errors, broken algorithms, wrong assumptions, off-by-one errors, type mismatches, contract violations, error propagation (swallowed errors, wrong error types).

2. **Regressions**: Removed behavior, changed contracts, broken integrations, data integrity issues, non-idempotent operations that should be idempotent.

3. **Security**: Injection (SQL, command, XSS), auth/authz issues, data exposure, hardcoded secrets/API keys, path traversal, SSRF, missing input validation, overly permissive CORS/permissions, new dependencies with known vulnerabilities.

4. **Performance**: N+1 queries, unnecessary loops, O(n²) when O(n) is possible, memory/resource leaks (connections, file handles), unnecessary allocations, missing indexes, blocking the event loop.

5. **Maintainability**: Naming, complexity, duplication, missing error handling, test coverage gaps.

6. **Edge Cases**: What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

If you're unsure whether something is an issue, put it in **Uncertain** rather than guessing.

## Severity Guide

- **Critical**: Will cause data loss, security breach, crash in production, or silent corruption. Must fix before commit.
- **High**: Bug or flaw that will bite someone, but won't cause immediate disaster. Should fix.
- **Low**: Code smell, minor inefficiency, or style issue that doesn't affect correctness. Consider fixing.

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

## Save Review

**IMPORTANT**: You MUST save the review. Determine the current branch name with `git branch --show-current`, replacing any `/` characters with `-` to keep it a flat filename. Run `mkdir -p _scratch/_reviews` then use the Write tool to write the full review output to `_scratch/_reviews/{branchname}-review.md`. Do not skip this step.
