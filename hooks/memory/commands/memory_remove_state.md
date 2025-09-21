---
name: Memory Remove State
description: Remove outdated state entries from memory
argument-hint: (interactive selection)
---

Interactively remove old state entries from the current ticket's memory.

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "Removing state for ticket: $TICKET (branch: $BRANCH)"
$HOME/.claude/hooks/memory/memory context remove state
```