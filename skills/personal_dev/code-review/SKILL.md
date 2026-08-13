---
name: code-review
description: Review committed changes on the current branch since it diverged from the base branch and report only issues that need fixing. Use when the user asks for a review of committed branch changes, a diff review against main, or whether the current branch is safe to merge. Default mode uses parallel sub-agent passes for correctness, security, performance, maintainability, and edge cases, then integrates a single findings-first review. Use `code-review single` only when the user explicitly asks for a single-pass review.
---

# Code Review

Review only the changes introduced on the current branch since merge-base. Use parallel sub-agents by default; use single-pass review only when the user or a calling skill explicitly requests `code-review single`.

## Scope

Review only commits between merge-base and `HEAD`, and only files this branch intentionally modified. Ignore unrelated pre-existing code, upstream changes brought in by merges or rebases, and rebase-noise files.

Use only the diff as evidence; put anything you are unsure about under `## Uncertain`.

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

## Parallel Review

Default mode. Run sub-agents in parallel, each working independently from the same diff.

If an implementation contract (Goal, Accepted scope, Assumptions, Out of scope, Validation approach) is provided in the caller's prompt, include it in each sub-agent's prompt so they review against the contract too. Treat Out of scope items as deliberate, settled exclusions.

- **Agent 1: Correctness & Regressions** — Does this code actually work? Logic errors, broken algorithms, wrong assumptions. Will merging break existing functionality? Removed behavior, changed contracts, broken integrations.
- **Agent 2: Security & Performance** — Injection risks, auth issues, data exposure, secrets in code. N+1 queries, unnecessary loops, memory leaks, expensive operations.
- **Agent 3: Maintainability & Edge Cases** — Naming, complexity, duplication, missing error handling, test coverage gaps. What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

## Single Review

Only when explicitly requested. Review focus:

1. **Correctness**: Logic errors, broken algorithms, wrong assumptions.
2. **Regressions**: Removed behavior, changed contracts, broken integrations.
3. **Security**: Injection risks, auth issues, data exposure, secrets in code.
4. **Performance**: N+1 queries, unnecessary loops, memory leaks, expensive operations.
5. **Maintainability**: Naming, complexity, duplication, missing error handling, test coverage gaps.
6. **Edge Cases**: Null handling, empty arrays, boundary conditions, race conditions.

## Output

List only issues that need fixing, each pointing at exactly what is wrong and where.

```
## Critical
Must fix before merge.
- [file:line] - [what is wrong and why it matters]

## High
Should fix.
- [file:line] - [what is wrong and why it matters]

## Low
Consider fixing.
- [file:line] - [what is wrong and why it matters]

## Uncertain
- [file:line] - [potential issue and why it is uncertain]

## Verdict
[1 sentence summary]
APPROVE | APPROVE WITH FIXES | NEEDS FIXES
```

End `## Verdict` with exactly one token on its own line: `APPROVE` (ready to merge as-is), `APPROVE WITH FIXES` (acceptable, specific fixes should land before merge), or `NEEDS FIXES` (still short of acceptable).

If no issues are found, say so plainly.

## Save Review

Also save a concise artifact. Branch name from `git branch --show-current`, replace `/` with `-`, `mkdir -p _scratch/_reviews`, then write `_scratch/_reviews/{branchname}-review.md`:

```
## Verdict
[APPROVE / APPROVE WITH FIXES / NEEDS FIXES]

## Blocking
[BLOCKING / NON-BLOCKING] - [1 short sentence on whether the findings are worth blocking over]

## Findings
- [Critical | High | Low | Uncertain] [file:line] - [what is wrong and why it matters]
```

With no findings: Verdict `APPROVE`, Blocking `NON-BLOCKING - No findings worth blocking over.`, Findings `- None.`
