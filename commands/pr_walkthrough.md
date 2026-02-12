---
description: Walk through a PR bottom-up, explaining each change and how it connects
version: "1.0"
---

# PR Walkthrough

Walk me through this PR starting from the base of the changes. Help me understand each step — what was added or changed, why it matters, and how the pieces connect.

This is a guided tour, not a code review. No suggestions. No judgments. Just clear explanation.

## Get Changes

```bash
# Current branch
BRANCH=$(git branch --show-current)

# Find the merge-base (where this branch diverged from main)
BASE=$(git merge-base origin/main HEAD 2>/dev/null || git merge-base origin/master HEAD)

# List commits on this branch (oldest first)
git log --oneline --no-merges --reverse $BASE..HEAD

# Full diff
git diff $BASE..HEAD

# Files changed summary
git diff --stat $BASE..HEAD
```

## Build the Walkthrough

### Step 1: Identify the Layers

Scan the diff and sort changes into layers:

1. **Foundations** — new types, interfaces, constants, utility functions. Things that don't call other new code.
2. **Core logic** — functions/methods that use the foundations. The actual work.
3. **Wiring** — registration, DI bindings, middleware hookups, config changes that connect new code to existing code.
4. **Entry points** — routes, handlers, CLI commands, event listeners, exports. Where the new behavior is triggered.

If a change doesn't fit neatly, place it where it first becomes relevant.

### Step 2: Walk Each Layer, Bottom-Up

For each layer (foundations → core logic → wiring → entry points), walk through the changes:

- **What changed**: name the function/type/file and what was added or modified.
- **Where**: `file:line` reference so the reader can jump directly to it.
- **Why it exists**: one sentence on the purpose. Connect it to the broader goal of the PR.
- **How it connects**: which other changes in this PR call it, depend on it, or are affected by it. Give `file:line` for each connection.

If commits are well-structured (each commit is a logical step), follow commit order within each layer. Otherwise, follow the dependency graph.

### Step 3: Trace Key Paths

After walking the layers, trace 1-3 key paths through the changes end-to-end:

> "When [trigger] happens, it hits [entry point] at `file:line`, which calls [core function] at `file:line`, which uses [foundation] at `file:line`."

Pick paths that best illustrate how the PR works as a whole.

## Output Format

```
## PR Walkthrough: {branch}

**Commits**: N commits on this branch
**Files changed**: N files

---

### Foundations

#### `functionOrTypeName` — short description
- **Where**: `path/to/file.go:42`
- **What**: Describe concretely what was added or changed.
- **Connects to**: `callerFunction` at `path/to/other.go:88`

#### ...

---

### Core Logic

#### `functionName` — short description
- **Where**: `path/to/file.go:100`
- **What**: Describe what this does.
- **Uses**: `foundationType` at `path/to/file.go:42`
- **Called by**: `handlerName` at `path/to/handler.go:55`

#### ...

---

### Wiring

#### ...

---

### Entry Points

#### ...

---

### Key Paths

1. **[Name the flow]**: `entryPoint` (`file:line`) → `coreFunction` (`file:line`) → `foundation` (`file:line`)

2. ...
```

Keep descriptions concrete. One to two sentences max per item. Let the `file:line` links do the heavy lifting — the reader will go look at the code.
