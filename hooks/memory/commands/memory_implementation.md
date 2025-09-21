---
name: Memory Implementation
description: Record implementation details to context (planning only)
argument-hint: [implementation description]
---

Record an implementation detail to memory context for the current ticket.

**IMPORTANT**: This command ONLY records the implementation to context for documentation.
It does NOT actually implement or execute anything. Use this to track what has been
implemented or what should be implemented.

Usage: Provide details about what was/will be implemented:
- Endpoints created (e.g., "POST /api/users")
- Functions added (e.g., "Added validateUser function")
- Features completed (e.g., "Implemented user authentication")
- Components built (e.g., "Created UserProfile component")

The implementation will be saved to context for future reference.

$ARGUMENTS

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

# Save to memory using verified ticket
$HOME/.claude/hooks/memory/memory context save implementation "$TICKET" "$ARGUMENTS"
```