---
name: code_review
description: Brutally honest review of committed code since branch diverged
version: "3.0"
---

# Code Review

Be brutally honest. Review ONLY the changes made on the current branch compared to main. Nothing else.

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

- **Critical**: Will cause data loss, security breach, crash in production, or silent corruption. Must fix before merge.
- **High**: Bug or flaw that will bite someone, but won't cause immediate disaster. Should fix.
- **Low**: Code smell, minor inefficiency, or style issue that doesn't affect correctness. Consider fixing.

## Output

List only issues that need fixing. No compliments. No padding.

```
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

## Save Review

**IMPORTANT**: You MUST save the review. Determine the current branch name with `git branch --show-current`, replacing any `/` characters with `-` to keep it a flat filename. Run `mkdir -p _scratch/_reviews` then use the Write tool to write the full review output to `_scratch/_reviews/{branchname}-review.md`. Do not skip this step.
