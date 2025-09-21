---
name: Memory Todo
description: Save a TODO item or blocker to memory
argument-hint: [TODO description or blocker details]
---

Save a TODO item, blocker, or next action to memory for the current ticket.

Usage: Provide details about what needs to be done, including:
- Description of the task or blocker
- Priority or urgency
- Dependencies or requirements
- Any context needed to complete it

$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save next "$ARGUMENTS"`