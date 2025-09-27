---
name: memory_get_todos
description: View all TODOs and next actions for the current branch
argument-hint: [optional filter keyword]
---

Display all TODO items and next actions saved in memory for the current branch/ticket.

Shows:
- Numbered TODO items for easy reference
- [COMPLETE] markers for finished items
- Quick overview without full context

Execute:
```bash
# Get actual current branch
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "TODOs for $TICKET (branch: $BRANCH)"
echo "======================================="
echo ""

# Use memory command to show just TODOs
$HOME/.claude/hooks/memory/memory context todos "$TICKET"
```