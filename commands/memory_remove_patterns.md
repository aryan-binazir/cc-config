---
name: Memory Remove Patterns
description: Remove outdated code patterns from memory
argument-hint: (item numbers to remove, e.g., "1,3" or "all", or leave blank to see list)
---

Remove outdated code patterns from the current ticket's memory.

If no arguments provided: Shows numbered list of patterns
If arguments provided: Removes those items (e.g., "1,3" removes items 1 and 3, "all" removes all)

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

echo "Patterns for ticket: $TICKET (branch: $BRANCH)"

# Pass arguments if provided, otherwise show list only
if [ -n "$ARGUMENTS" ]; then
    $HOME/.claude/hooks/memory/memory context remove patterns "$ARGUMENTS"
else
    # Just show the list, no removal
    $HOME/.claude/hooks/memory/memory context remove patterns | head -n -1
    echo ""
    echo "To remove items, run: /memory_remove_patterns 1,3  (or 'all' to remove all)"
fi
```