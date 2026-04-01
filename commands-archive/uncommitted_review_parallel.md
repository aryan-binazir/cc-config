---
name: uncommitted_review_parallel
description: Parallel review of uncommitted changes
version: "2.0"
---

# Uncommitted Review (Parallel)

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

## Parallel Review

Spawn 3 sub-agents in parallel. Each agent must **read the full changed files for context** — do not review the diff in isolation.

Only flag real issues. Do not stretch to fill categories.

**Agent 1: Correctness & Regressions**
- Logic errors, broken algorithms, wrong assumptions
- Off-by-one errors, type mismatches, contract violations
- Swallowed errors, wrong error types, missing error propagation
- Will committing break existing functionality?
- Removed behavior, changed contracts, data integrity, idempotency

**Agent 2: Security & Performance**
- Injection (SQL, command, XSS), auth/authz issues
- Data exposure, hardcoded secrets, path traversal, SSRF
- Missing input validation, overly permissive CORS/permissions
- N+1 queries, O(n²) when O(n) is possible
- Memory/resource leaks, unnecessary allocations
- Missing indexes, blocking the event loop

**Agent 3: Maintainability & Edge Cases**
- Naming, complexity, duplication, missing error handling
- Test coverage gaps
- Null handling, empty arrays, boundary conditions, race conditions

Each agent: be specific with file:line references. If unsure, say so — put it in Uncertain.

## Integration

After all agents report back:
1. **Deduplicate** — if multiple agents flagged the same issue, keep the best description
2. **Classify severity** using this guide:
   - **Critical**: Will cause data loss, security breach, crash in production, or silent corruption
   - **High**: Bug or flaw that will bite someone, but won't cause immediate disaster
   - **Low**: Code smell, minor inefficiency, or style issue that doesn't affect correctness
3. **Synthesize** into a single review using the output format below

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

After outputting the review, also write the full review output to `_scratch/REVIEW.md`. If `REVIEW.md` already exists, increment the suffix: `REVIEW_1.md`, `REVIEW_2.md`, etc.
