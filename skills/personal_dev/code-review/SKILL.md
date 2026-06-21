---
name: code-review
description: Review committed changes on the current branch since it diverged from the base branch and report only issues that need fixing. Default mode uses parallel sub-agent passes for correctness, security, performance, maintainability, and edge cases, then integrates a single findings-first review. Use `code-review single` only when the user explicitly asks for a single-pass review.
---

# Code Review

Review only the changes introduced on the current branch since merge-base. Use parallel sub-agents by default, and only use single-pass review when the user explicitly asks for `code-review single`.

## Scope

Review only:
- commits between merge-base and `HEAD`
- files actually modified by this branch

Do not review:
- unrelated pre-existing code
- upstream changes brought in by merges or rebases
- files that appear only due to rebase noise and were not intentionally changed on the branch

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

If a file appears in the diff that wasn't intentionally modified on this branch, ignore it -- it's likely a rebase artifact.

## Parallel Review

Use this mode by default.

If an implementation contract (Goal, Accepted scope, Assumptions, Out of scope, Validation approach) is provided in the caller's prompt, include it in each sub-agent's prompt so they review against the contract too. Respect Out of scope items — do not treat them as missing work.

- **Agent 1: Correctness & Regressions** -- Does this code actually work? Logic errors, broken algorithms, wrong assumptions. Will merging break existing functionality? Removed behavior, changed contracts, broken integrations.
- **Agent 2: Security & Performance** -- Injection risks, auth issues, data exposure, secrets in code. N+1 queries, unnecessary loops, memory leaks, expensive operations.
- **Agent 3: Maintainability & Edge Cases** -- Naming, complexity, duplication, missing error handling, test coverage gaps. What inputs would break this? Null handling, empty arrays, boundary conditions, race conditions.

Be specific. Point out exactly what's wrong and where. No padding.

Only use information from the diff. If you're unsure whether something is an issue, say so rather than guessing.

## Single Review

Use this mode only when the user explicitly asks for `code-review single` or a caller explicitly requests single-pass review.

Review focus:

1. **Correctness**: Logic errors, broken algorithms, wrong assumptions.
2. **Regressions**: Removed behavior, changed contracts, broken integrations.
3. **Security**: Injection risks, auth issues, data exposure, secrets in code.
4. **Performance**: N+1 queries, unnecessary loops, memory leaks, expensive operations.
5. **Maintainability**: Naming, complexity, duplication, missing error handling, test coverage gaps.
6. **Edge Cases**: Null handling, empty arrays, boundary conditions, race conditions.

Only use information from the diff. If you're unsure whether something is an issue, say so rather than guessing.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Critical (must fix before merge)
- [file:line] - [what is wrong and why it matters]

## High (should fix)
- [file:line] - [what is wrong and why it matters]

## Low (consider fixing)
- [file:line] - [what is wrong and why it matters]

## Uncertain
- [file:line] - [potential issue and why it is uncertain]

## Verdict
[APPROVE / NEEDS FIXES / REJECT] - [1 sentence summary]
```

If no issues are found, say so plainly.

## Save Review

Also save a concise artifact to `_scratch/_reviews/{branchname}-review.md` using this format:

```
## Verdict
[APPROVE / NEEDS FIXES / REJECT]

## Blocking
[BLOCKING / NON-BLOCKING] - [1 short sentence on whether the findings are worth blocking over]

## Findings
- [Critical | High | Low | Uncertain] [file:line] - [what is wrong and why it matters]
```

If there are no findings, write:

```
## Verdict
APPROVE

## Blocking
NON-BLOCKING - No findings worth blocking over.

## Findings
- None.
```

1. Determine the current branch name with `git branch --show-current`.
2. Replace any `/` characters with `-` so the filename stays flat.
3. Run `mkdir -p _scratch/_reviews`.
4. Write the review to `_scratch/_reviews/{branchname}-review.md`.
