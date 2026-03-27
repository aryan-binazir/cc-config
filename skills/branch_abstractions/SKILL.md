---
name: branch_abstractions
description: Analyze the current branch diff, identify the non-obvious abstractions introduced or used in the change, resolve where they live in the codebase, and write a short explainer artifact. Use when the user wants help understanding branch-specific abstractions, indirection, wiring, factories, registries, or unfamiliar symbols introduced on the current branch.
---

# Branch Abstractions

Analyze only the changes on the current branch compared with its base branch and produce a compact explainer for the abstractions that matter.

## Scope

Review only the branch diff from merge-base to `HEAD`.

Do not explain unrelated abstractions that are not touched by the diff unless they are the minimum context needed to understand a changed abstraction.

## Workflow

1. Determine the current branch name.
2. Find the merge-base against `origin/main`, falling back to `origin/master` if needed.
3. Inspect the branch diff and diff summary.
4. Extract abstraction candidates from added or modified lines. Prioritize:
   - qualified symbols such as `pkg.Type.Method`, `svc.DoThing`, or `module.sub.fn`
   - builders and chains such as `NewX(...).WithY(...).Build()`
   - registration and wiring such as middleware, plugin registries, hooks, DI bindings, or route registration
   - non-trivial types appearing in signatures, struct fields, or constructor parameters
   - factories and constructors such as `NewFoo`, `createBar`, or `BuildBaz`
5. Skip trivial references such as standard-library calls, obvious local variables, or string literals.
6. De-duplicate aggressively so each abstraction appears once.
7. Resolve each abstraction with fast repo searches. Record a confident definition path and line when found. If not, mark it unresolved and include the best search pointer.
8. Write an artifact to `_scratch/_context/{branch}-abstracts.md`.

## Artifact Format

Use this structure:

```md
# Abstractions: {branch}

- **Branch**: `{branch}`
- **Diff**: `{BASE}..HEAD`
- **Files analyzed**: N

## Abstractions Index

| Name | Kind | Defined at | Used in diff |
|------|------|-----------|--------------|

## Notes

### `Name`
- **What**: concrete one-line explanation
- **Why**: one or two bullets explaining why it exists
- **Usage in this diff**: tie to changed lines
- **Read next**: definition path and one or two good follow-ups
```

Keep each abstraction note to about 3 to 6 bullets. Be concrete and avoid speculation.

## Response

After writing the artifact, print:

```
Abstractions artifact: _scratch/_context/{branch}-abstracts.md
Abstractions found: N
Top abstractions:
  - Name1
  - Name2
  - Name3
```

Show up to 5 names. Keep the terminal response short.
