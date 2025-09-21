---
name: Memory Sync
description: Sync current work to memory using git diff and branch context
argument-hint: [optional additional context notes]
---

Capture and save the current state of work to memory by syncing git changes and providing context.

Steps:
1. Extract the current ticket from git branch name
2. Capture git diff of staged and unstaged changes
3. Get current git status for context
4. Combine with any additional notes from $ARGUMENTS
5. Save everything to memory as 'state' category

Execute:
```bash
# Get current branch/ticket
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$(echo "$BRANCH" | grep -oE '[A-Z]+-[0-9]+' || echo "no-ticket")

# Capture current state
STATUS=$(git status --porcelain 2>/dev/null || echo "No git status available")
DIFF=$(git diff HEAD 2>/dev/null || echo "No diff available")

# Build context message
CONTEXT="Branch: $BRANCH

Git Status:
$STATUS

Changes:
$DIFF"

# Add user arguments if provided
if [ -n "$ARGUMENTS" ]; then
    CONTEXT="$CONTEXT

Additional Notes:
$ARGUMENTS"
fi

# Save to memory
$HOME/.claude/hooks/memory/memory context save state "$CONTEXT"
```