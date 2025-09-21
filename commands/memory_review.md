---
name: Memory Review
description: Review and display current ticket's memory context
argument-hint: [optional filter or search terms]
---

Review the saved memory context for the current ticket.

This command retrieves and displays all saved memory entries (decisions, implementations, patterns, state, and next actions) for the current ticket extracted from the git branch.

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "🔍 Reviewing memory for ticket: $TICKET"
echo "Current branch: $BRANCH"
echo ""

# Query memory for this ticket
$HOME/.claude/hooks/memory/memory context query "$TICKET" "$ARGUMENTS"
```