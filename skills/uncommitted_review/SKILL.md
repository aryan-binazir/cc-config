---
name: uncommitted_review
description: Review only the staged and unstaged uncommitted changes in the working tree and report only issues that need fixing. Use when the user asks for a review of current local changes before commit.
---

# Uncommitted Review

Review only the working tree changes that are not yet committed.

## Scope

Review only:
- staged changes from `git diff --cached`
- unstaged changes from `git diff`

Do not review already committed code or unrelated files.

## Review Focus

1. Correctness
2. Regressions
3. Security
4. Performance
5. Maintainability
6. Edge cases

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
