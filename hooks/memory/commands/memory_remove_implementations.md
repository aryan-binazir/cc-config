---
name: Memory Remove Implementations
description: Remove outdated implementations from memory
argument-hint: (interactive selection)
---

Interactively remove old implementation notes from the current ticket's memory.

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "Removing implementations for ticket: $TICKET (branch: $BRANCH)"
$HOME/.claude/hooks/memory/memory context remove implementations
```