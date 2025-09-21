---
name: Memory Decision
description: Save architectural or technical decisions to memory
argument-hint: [decision description and rationale]
---

Save an architectural or technical decision to memory for the current ticket.

Usage: Provide the decision details as arguments, including:
- What was decided
- Why it was decided (rationale)
- Any alternatives considered
- Impact or implications

$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save decision "$ARGUMENTS"`