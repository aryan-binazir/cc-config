---
description: Analyze current branch diff and map abstractions introduced or used
version: "1.0"
---

# Branch Abstractions

Analyze the current branch's changes, extract abstractions introduced or used in the diff, resolve where they live, and write a short explainer artifact.

## Get Changes

```bash
# Current branch
BRANCH=$(git branch --show-current)

# Find the merge-base (where this branch diverged from main)
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)

# Get ONLY the diff between merge-base and current HEAD
git diff $BASE..HEAD

# Summary of files changed on this branch
git diff --stat $BASE..HEAD
```

## Extract Abstraction Candidates

Focus on added/modified lines from the diff. Prioritize non-obvious indirections:

- **Qualified symbols**: `A.B.C`, `pkg.Type.Method`, `svc.DoThing`, `module.sub.fn`
- **Builders/chains**: `NewX(...).WithY(...).Build()`
- **Wiring/registration**: DI container bindings, middleware/interceptors, plugin registries, hook registrations
- **New domain/service/repo types** showing up in function signatures, struct fields, or constructor params
- **Factories and constructors**: `NewFoo(...)`, `createBar(...)`, `BuildBaz(...)`

Skip trivial references (standard library calls, simple variable access, obvious string literals). De-dupe aggressively — one entry per unique abstraction.

## Resolve Each Abstraction (Best Effort)

For each candidate:

1. Search the repo for its definition using `rg` or language-aware patterns typical for the repo's languages.
2. If confidently found: record the file path and line number.
3. If not confidently resolved: mark as "unresolved" and include the best search pointer (e.g., likely package, likely file pattern).

Keep resolution fast — only search for detected abstractions, not full-repo scans.

## Write Artifact

- Store in `context/` directory (create if needed).
- Filename: `context/{branch}-abstracts.md` where `{branch}` is the current git branch name.

### Artifact Format

```md
# Abstractions: {branch}

- **Branch**: `{branch}`
- **Diff**: `{BASE}..HEAD`
- **Files analyzed**: N

## Abstractions Index

| Name | Kind | Defined at | Used in diff |
|------|------|-----------|--------------|
| `pkg.Type.Method` | method | `path/to/file.go:42` | `changed/file.go:15,28` |
| `NewFoo` | factory | unresolved (likely `pkg/foo/`) | `changed/file.go:7` |

## Notes

### `pkg.Type.Method`
- **What**: (concrete 1-liner: what it is/does)
- **Why**: (1-2 bullets: why it exists)
- **Usage in this diff**: (tie to changed lines)
- **Read next**: definition at `path:line` + 1-2 call sites or tests

### `NewFoo`
- **What**: ...
- **Why**: ...
- **Usage in this diff**: ...
- **Read next**: ...
```

Limit notes to 3-6 bullets per abstraction. Be concrete, not speculative.

## Stdout Summary

After writing the artifact, print:

```
Abstractions artifact: context/{branch}-abstracts.md
Abstractions found: N
Top abstractions:
  - Name1
  - Name2
  - Name3
  - Name4
  - Name5
```

Show up to 5. Keep it short.
