---
name: code-review
description: Review only the committed changes on the current branch since it diverged from the base branch and report only issues that need fixing. Use when the user asks for a review of committed branch changes, a diff review against main, or whether the current branch is safe to merge.
---

# Code Review

Review only the changes introduced on the current branch compared with its merge-base against the default branch.

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

## Review Focus

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
