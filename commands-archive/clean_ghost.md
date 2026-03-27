---
name: clean_ghost
description: Remove all ghost implementation comments from the codebase
argument-hint: "[optional: specific file or directory path]"
version: "1.0"
---

Remove all ghost implementation scaffolding from source files.

## Scope

$ARGUMENTS

- If a file path: Clean only that file
- If a directory path: Clean all files in that directory (recursively)
- If empty: Clean the entire codebase (from repo root)

## What Gets Removed

Ghost comments follow this pattern and should be completely removed:

```
// GHOST: <name>
// WHY: <...>
// IN:  <...>
// OUT: <...>
// SAMPLE: <...> (optional, may span multiple lines)
<function stub that throws/raises "not implemented">
```

**Detection patterns:**

1. Comment lines starting with `GHOST:`, `WHY:`, `IN:`, `OUT:`, `SAMPLE:`
2. Function stubs containing:
   - `throw new Error("Not implemented")`
   - `raise NotImplementedError()`
   - `errors.New("not implemented")`
   - `panic("not implemented")`
   - `// TODO: implement` immediately after ghost comments

## Workflow

### Step 1: Find Ghost Comments

Search for files containing `GHOST:` comments:
```
grep -r "GHOST:" --include="*.ts" --include="*.js" --include="*.py" --include="*.go" --include="*.rs" --include="*.java" .
```

### Step 2: Review Before Removal

List all ghost comments found:

```
## Ghost Comments Found

- `path/to/file.ts:45` - functionName
- `path/to/other.ts:120` - otherFunction
- ...

Total: X ghost comments in Y files
```

Ask user to confirm: "Remove all ghost comments? (y/n)"

### Step 3: Remove Ghost Blocks

For each ghost comment block:
1. Remove the entire GHOST comment block (GHOST, WHY, IN, OUT, SAMPLE lines)
2. Remove the associated stub function/method
3. If removing leaves an empty file, delete the file
4. If removing leaves orphaned imports, remove those too

### Step 4: Report

```
## Ghost Cleanup Complete

Removed X ghost comments from Y files:

- `path/to/file.ts` - removed: functionName
- `path/to/other.ts` - removed: otherFunction
- ...

Files deleted (were ghost-only):
- `path/to/new-file.ts`
```

## Rules

- Always show what will be removed before removing
- Require confirmation before deletion
- Don't remove comments that merely mention "ghost" in regular context
- Only remove the specific GHOST block pattern, not surrounding code
- Preserve file if it has non-ghost code remaining
