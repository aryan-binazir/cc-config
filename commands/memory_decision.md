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

Run: `$HOME/.claude/hooks/memory/memory context save decision "$ARGUMENTS"`