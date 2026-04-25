---
name: clean-ghost
description: Remove ghost implementation scaffolding comments and their associated placeholder stubs from a file, directory, or repository after previewing exactly what would be removed. Use when the user asks to clean ghost comments, remove implementation scaffolding, or strip `GHOST:` blocks after real implementation is done.
---

# Clean Ghost

Remove ghost implementation scaffolding from source files, but do not remove anything until you have shown the user exactly what will be deleted and they have confirmed.

## Scope Selection

Infer scope from the request:
- if the user names a file, operate on that file only
- if the user names a directory, operate recursively within that directory
- if no scope is provided, treat the repo root as the scope

## What Counts As Ghost Scaffolding

Ghost blocks follow this pattern:

```
// GHOST: <name>
// WHY: <...>
// IN:  <...>
// OUT: <...>
// SAMPLE: <...>   optional and may span multiple lines
<function stub that throws or raises "not implemented">
```

Detection rules:
- comment lines beginning with `GHOST:`, `WHY:`, `IN:`, `OUT:`, or `SAMPLE:`
- associated placeholder stubs such as `throw new Error("Not implemented")`, `raise NotImplementedError()`, `errors.New("not implemented")`, `panic("not implemented")`, or a nearby `TODO: implement`

## Workflow

1. Search only the chosen scope for `GHOST:` blocks.
2. Build a preview list showing file paths, line numbers, and the ghosted symbol names.
3. Present a removal summary to the user and ask for confirmation before deleting anything.
4. After confirmation, remove:
   - the entire ghost comment block
   - the associated stub function or method
   - orphaned imports left behind by that removal
5. If a file becomes empty because it contained only ghost scaffolding, delete the file.
6. Report what was removed and which files, if any, were deleted.

## Rules

- Always preview before removal.
- Always require confirmation before deleting.
- Do not remove comments that merely mention the word "ghost" in normal prose.
- Remove only the explicit ghost block pattern and its associated placeholder stub.
- Preserve any remaining functional code in the file.

## Output

Before confirmation, use:

```
## Ghost Comments Found
- `path/to/file.ts:45` - functionName
Total: X ghost comments in Y files
```

After cleanup, use:

```
## Ghost Cleanup Complete
Removed X ghost comments from Y files:
- `path/to/file.ts` - removed: functionName

Files deleted:
- `path/to/ghost_only_file.ts`
```
