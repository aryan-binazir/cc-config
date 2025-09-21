---
name: Memory Remove Decisions
description: Remove outdated decisions from memory
argument-hint: (interactive selection)
---

Interactively remove outdated architectural decisions from the current ticket's memory.

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "Removing decisions for ticket: $TICKET (branch: $BRANCH)"
$HOME/.claude/hooks/memory/memory context remove decisions
```