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
   - `Plan` → update if approach changes
   - `Tasks` → check off completed, add new ones
   - `Useful Information` → append learnings, gotchas, references
   - `Decisions` → append any new decisions made
5. Leave stable sections unchanged: Summary (unless explicitly discussed)
6. Reply with 1-3 bullet summary of changes (mention which file was updated)

## Template (only if creating new context file)

```md
# Project Context

Last updated: YYYY-MM-DD

## Summary
*(1 paragraph: what this project is and why it exists.)*

## Plan
*(Original plan and approach. Update as decisions change.)*
- *(Step/milestone)*

## Tasks
- [ ] *(Task to complete)*
- [x] *(Completed task)*

## Useful Information
*(Things learned along the way that provide helpful context.)*
- *(Insight, gotcha, or reference)*

## Decisions
- `YYYY-MM-DD`: *(Decision + brief rationale)*
```
