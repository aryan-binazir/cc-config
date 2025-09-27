---
name: mark_todo_complete
description: Mark a specific TODO as complete by adding [COMPLETE] prefix
argument-hint: <todo number from /memory_review>
---

Mark a TODO item as complete when you've finished implementing it.

Steps:
1. Takes the TODO number shown in /memory_review output
2. Adds [COMPLETE] prefix to that specific TODO
3. Preserves the TODO in memory for reference

Execute:
```bash
# Get actual current branch
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

# Validate TODO number provided
if [ -z "$ARGUMENTS" ]; then
    echo "Error: Please provide the TODO number to mark complete"
    echo "Usage: /mark_todo_complete <number>"
    echo "Get TODO numbers from /memory_review output"
    exit 1
fi

# Mark the specific TODO as complete
$HOME/.claude/hooks/memory/memory context mark-complete "$TICKET" "$ARGUMENTS"
```