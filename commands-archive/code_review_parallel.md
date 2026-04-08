---
name: code_review_parallel
description: Parallel review of committed code since branch diverged
version: "2.0"
---

# Code Review (Parallel)

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

## Parallel Review

Spawn 3 sub-agents in parallel. Each agent must **read the full changed files for context** — do not review the diff in isolation.

Only flag real issues. Do not stretch to fill categories.

**Agent 1: Correctness & Regressions**
- Logic errors, broken algorithms, wrong assumptions
- Off-by-one errors, type mismatches, contract violations
- Swallowed errors, wrong error types, missing error propagation
- Will merging break existing functionality?
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

**IMPORTANT**: You MUST save the review. Determine the current branch name with `git branch --show-current`. Run `mkdir -p _scratch/_reviews` then use the Write tool to write the full review output to `_scratch/_reviews/{branchname}-review.md` (replacing `{branchname}` with the actual branch name). Do not skip this step.
