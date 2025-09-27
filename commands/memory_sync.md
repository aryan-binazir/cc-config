---
name: Memory Sync
description: Sync current work to memory using git diff and branch context
argument-hint: [optional additional context notes]
---

Capture and save the current state of work to memory by extracting code patterns from git diff.

Steps:
1. Extract code patterns (functions, types, interfaces) from git diff
2. Save any additional notes provided as implementation context

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

# Extract and save code patterns from git diff
echo "Syncing code patterns from git diff..."
$HOME/.claude/hooks/memory/memory context sync-git

# If additional notes provided, save as implementation
if [ -n "$ARGUMENTS" ]; then
    echo "Saving additional context notes..."
    $HOME/.claude/hooks/memory/memory context save implementation "$TICKET" "$ARGUMENTS"
fi

echo "Memory sync complete for $TICKET"
```