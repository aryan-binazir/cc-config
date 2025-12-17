---
name: context_sync
description: Sync current project state into CONTEXT.md
argument-hint: "[optional notes to include]"
---

Update or create the branch-specific context file to reflect current project state. This is the single source of truth for the project's plan, assumptions, and status.

## File Naming

- Get current branch: `git branch --show-current`
- Store all context files in `context/` directory (create if needed)
- If branch is `main` or `master`: use `context/CONTEXT.md`
- Otherwise: use `context/CONTEXT-{branch}.md` (e.g., `context/CONTEXT-feature-auth.md`)

## Rules

- Be factual - don't invent progress
- Preserve existing valuable context
- Never include secrets (tokens, API keys, credentials)
- Keep it skimmable (bullets + checklists)

## Process

1. Determine context file name based on current branch
2. Check if context file exists:
   - **If new**: Create from template below, populate based on conversation
   - **If exists**: Read current file, then selectively update (see step 4)
3. Check `git status -sb` and `git diff --stat` for repo state
4. Update these sections based on conversation and `$ARGUMENTS`:
   - `Last updated` → today's date
   - `Current Objective` → if focus has shifted
   - `Next Up` → add new tasks, move completed ones to Completed
   - `Completed` → add finished items with date
   - `Decisions` → append any new decisions made
   - `Notes` → append relevant info
5. Leave stable sections unchanged: Summary, Assumptions (unless explicitly discussed)
6. Reply with 1-3 bullet summary of changes (mention which file was updated)

## Template (only if creating new context file)

```md
# Project Context

Last updated: YYYY-MM-DD

## Summary
*(1 paragraph: what this project is and why it exists.)*

## Current Objective
*(What we’re doing right now; 1–3 bullets.)*

## Assumptions & Constraints
- *(Assumption/constraint)*

## Decisions
- `YYYY-MM-DD`: *(Decision + brief rationale)*

## High-Level Plan
- *(Milestone 1)*
- *(Milestone 2)*

## Next Up
- [ ] *(Concrete next task)*
- [ ] *(Concrete next task)*

## Completed
- `YYYY-MM-DD`: *(What was completed)*

## Notes
- *(Anything that doesn’t fit elsewhere)*
```
