---
name: code-review-ar
description: Findings-only review of the current branch's committed changes since merge-base, ending in a merge verdict. Use to review a branch, diff it against main, or judge whether it is safe to merge. Runs parallel sub-agents; `code-review-ar single` runs one pass, only on explicit request.
---

# Code Review AR

## Scope

Review only commits between merge-base and `HEAD`, and only files this branch intentionally modified; pre-existing code, upstream changes pulled in by merges or rebases, and rebase noise stay out. Judge the diff against repo rules (`AGENTS.md`, `CLAUDE.md`, coding standards), surrounding conventions, and any supplied implementation contract and its scope. Ground every finding in what the diff introduced; put doubts under `## Uncertain`.

## Get Changes

```bash
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)
git diff $BASE..HEAD
git log --oneline --no-merges $BASE..HEAD
git diff --stat $BASE..HEAD
```

## Lenses

1. **Correctness**: logic errors, broken algorithms, wrong assumptions.
2. **Regressions**: removed behavior, changed contracts, broken integrations.
3. **Security**: injection, auth, data exposure, secrets in code.
4. **Performance**: N+1 queries, needless loops, memory leaks, expensive operations.
5. **Maintainability**: repo rules and conventions, naming, complexity, duplication, missing error handling, test coverage gaps.
6. **Edge cases**: null handling, empty arrays, boundary conditions, race conditions.

**Parallel** (default): three independent sub-agents on the same diff, each handed any contract and the applicable repo rules, covering lenses 1–2, 3–4, and 5–6. Integrate their findings into one review.

**Single** (`code-review-ar single`, only when the user or a calling skill asks): one pass over all six lenses.

## Output

List only issues that need fixing, each pinned to exactly what is wrong and where. With none, say so plainly.

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

End `## Verdict` with exactly one token on its own line: `APPROVE` (merge as-is), `APPROVE WITH FIXES` (acceptable once specific fixes land), or `NEEDS FIXES` (not yet acceptable).

## Save Review

Also write a concise artifact to `_scratch/_reviews/<branch>-review.md`, where `<branch>` is `git branch --show-current` with `/` → `-`:

```
## Verdict
[APPROVE / APPROVE WITH FIXES / NEEDS FIXES]

## Blocking
[BLOCKING / NON-BLOCKING] - [1 short sentence on whether the findings are worth blocking over]

## Findings
- [Critical | High | Low | Uncertain] [file:line] - [what is wrong and why it matters]
```

With no findings: Verdict `APPROVE`, Blocking `NON-BLOCKING - No findings worth blocking over.`, Findings `- None.`
