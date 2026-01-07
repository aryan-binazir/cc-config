---
description: Brutally honest review of uncommitted changes for correctness and regressions
version: "1.0"
---

# Review Uncommitted Changes

ultrathink

Review all uncommitted changes (staged and unstaged) for correctness and potential regressions. Be brutally honest.

## Get Changes

```bash
git diff HEAD          # All uncommitted changes
git diff --stat HEAD   # Summary
```

## Review Focus

1. **Correctness**: Does this code actually do what it's supposed to? Logic errors, off-by-ones, wrong conditions, broken edge cases.

2. **Regressions**: Will this break existing functionality? Changed behavior, removed code that was needed, broken contracts.

3. **Bugs introduced**: Null pointer risks, unhandled errors, race conditions, resource leaks.

## Output

List only issues that need fixing. No compliments. No padding.

```
## Issues

1. [file:line] - [what's wrong and why it matters]
2. [file:line] - [what's wrong and why it matters]
...

## Verdict
[GOOD TO COMMIT / NEEDS FIXES] - [1 sentence summary]
```

If no issues found, say so and move on.
