---
name: Memory Implementation
description: Record implementation details to context (documentation only)
argument-hint: [implementation description and details]
---

Record implementation details to memory context for the current ticket.

**IMPORTANT**: This command is for DOCUMENTATION purposes only. It records what
has been implemented or what should be implemented, but does NOT execute any
implementation work. Use this to track completed work or planned implementations
for future reference.

Usage: Provide details about what was/will be implemented, including:
- What was built or needs to be built
- How it works or should work
- Key components or files modified/to modify
- Any important implementation notes

The implementation details will be saved to context for documentation and planning.

$ARGUMENTS

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

# Save to memory using verified ticket
$HOME/.claude/hooks/memory/memory context save implementation "$TICKET" "$ARGUMENTS"
```