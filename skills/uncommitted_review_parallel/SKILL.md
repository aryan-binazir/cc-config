---
name: uncommitted_review_parallel
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

Keep the review concise. No compliments. No padding.

```
## Verdict
[GOOD TO COMMIT / NEEDS FIXES]

## Blocking
[BLOCKING / NON-BLOCKING] - [1 short sentence]

## Findings
- [Critical | High | Low | Uncertain] [file:line] - [what is wrong and why it matters]
```

If no issues are found, say so plainly:

```
## Verdict
GOOD TO COMMIT

## Blocking
NON-BLOCKING - No findings worth blocking over.

## Findings
- None.
```

## Save Review

Save exactly the same concise review output shown in the terminal.

1. Determine the current branch name with `git branch --show-current`.
2. Replace any `/` characters with `-` so the filename stays flat.
3. Run `mkdir -p _scratch/_reviews`.
4. Write the review to `_scratch/_reviews/{branchname}-review.md`.
