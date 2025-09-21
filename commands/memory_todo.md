---
name: Memory Todo
description: Record a TODO item or blocker to context (no auto-execution)
argument-hint: [TODO description or blocker details]
---

Record a TODO item, blocker, or next action to memory context for the current ticket.

**IMPORTANT**: This command ONLY records the task to context for tracking purposes.
It does NOT automatically execute or start working on the task. The task will be
added to your project context for later reference and planning.

Usage: Provide details about what needs to be done, including:
- Description of the task or blocker
- Priority or urgency
- Dependencies or requirements
- Any context needed to complete it

The task will be saved to memory for future reference, not immediately executed.

$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save next "$ARGUMENTS"`