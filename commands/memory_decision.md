---
name: Memory Decision
description: Record architectural/technical decisions to context (planning only)
argument-hint: [decision description and rationale]
---

Record an architectural or technical decision to memory context for the current ticket.

**IMPORTANT**: This command ONLY records the decision to context for documentation
and future reference. It does NOT implement or act on the decision. Use this to
track important architectural choices and design decisions that have been made or
need to be considered.

Usage: Provide the decision details as arguments, including:
- What was/should be decided
- Why it was decided (rationale)
- Any alternatives considered
- Impact or implications

The decision will be saved to context for documentation and future reference.

$ARGUMENTS

Execute:
```bash
# Get actual current branch - don't trust context
BRANCH=$(git branch --show-current 2>/dev/null || echo "no-branch")
TICKET=$($HOME/.claude/hooks/memory/memory extract-ticket "$BRANCH" 2>/dev/null || echo "$BRANCH")

# Save to memory using verified ticket
$HOME/.claude/hooks/memory/memory context save decision "$TICKET" "$ARGUMENTS"
```