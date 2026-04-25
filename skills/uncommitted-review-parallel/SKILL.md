---
name: uncommitted-review-parallel
description: Review staged and unstaged working-tree changes with parallel sub-agent passes for correctness, security, performance, maintainability, and edge cases, then integrate a single findings-first review. Use when the user wants a deeper or parallel review of current uncommitted changes.
---

# Uncommitted Review Parallel

Review only staged and unstaged changes in the working tree, using parallel passes only when they can operate independently from the same diff.

## Scope

Same scope as `uncommitted_review`: only `git diff --cached` and `git diff`.

## Workflow

1. Gather staged and unstaged diffs plus a short status summary.
2. Split the review into parallel passes:
   - correctness and regressions
   - security and performance
   - maintainability and edge cases
3. Integrate findings into one non-duplicative review.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Critical (must fix before commit)
- [file:line] - [what is wrong and why it matters]

## High (should fix)
- [file:line] - [what is wrong and why it matters]

## Low (consider fixing)
- [file:line] - [what is wrong and why it matters]

## Uncertain
- [file:line] - [potential issue and why it is uncertain]

## Verdict
[GOOD TO COMMIT / NEEDS FIXES] - [1 sentence summary]
```

If no issues are found, say so plainly.

## Save Review

Also save a concise artifact to `_scratch/_reviews/{branchname}-review.md` using this format:

```
## Verdict
[GOOD TO COMMIT / NEEDS FIXES]

## Blocking
[BLOCKING / NON-BLOCKING] - [1 short sentence on whether the findings are worth blocking over]

## Findings
- [Critical | High | Low | Uncertain] [file:line] - [what is wrong and why it matters]
```

If there are no findings, write:

```
## Verdict
GOOD TO COMMIT

## Blocking
NON-BLOCKING - No findings worth blocking over.

## Findings
- None.
```

1. Determine the current branch name with `git branch --show-current`.
2. Replace any `/` characters with `-` so the filename stays flat.
3. Run `mkdir -p _scratch/_reviews`.
4. Write the review to `_scratch/_reviews/{branchname}-review.md`.
