---
name: Memory Review
description: Review and display current ticket's memory context
argument-hint: [optional filter or search terms]
---

Review the saved memory context for the current ticket.

This command retrieves and displays all saved memory entries (decisions, implementations, patterns, state, and next actions) for the current ticket extracted from the git branch.

Execute:
```bash
# Get current ticket from branch
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$(echo "$BRANCH" | grep -oE '[A-Z]+-[0-9]+' || echo "no-ticket")

echo "🔍 Reviewing memory for ticket: $TICKET"
echo "Current branch: $BRANCH"
echo ""

# Query memory for this ticket
$HOME/.claude/hooks/memory/memory context query "$TICKET" "$ARGUMENTS"
```