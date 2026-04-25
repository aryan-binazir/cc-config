---
name: context-sync
description: Update or create the branch-specific project context file under `_scratch/_context/` so it reflects the current plan, status, decisions, and repo state. Use when the user asks to sync project context, update branch notes, capture current status, or maintain a branch context file.
---

# Context Sync

Maintain a branch-specific context file as the single source of truth for the current project state.

## File Naming

- Use the current git branch name.
- Store files in `_scratch/_context/`.
- File name: `_scratch/_context/{branch}.md`, replacing `/` in branch names with `-`.

## Rules

- Be factual. Do not invent progress.
- Preserve existing useful context.
- Never include secrets.
- Keep the file skimmable with short paragraphs, bullets, and checklists.

## Workflow

1. Determine the branch-specific context file path.
2. If it exists, read it before editing. If it does not exist, create it from the template below.
3. Check repo state with `git status -sb` and a diff summary.
4. Update these sections using the current conversation and any user-provided notes:
   - `Status`
   - `Links`
   - `Last updated`
   - `Plan`
   - `Tasks`
   - `Useful Information`
   - `Decisions`
5. Leave stable sections unchanged unless the conversation materially changed them.
6. Reply with a short summary of what changed and which file was updated.

## Template

```md
# Project Context

Status: In Progress

## Links
- Jira: *(link or N/A)*
- GitHub PR: *(link or N/A)*

Last updated: YYYY-MM-DD

## Summary
*(1 paragraph: what this project is and why it exists.)*

## Plan
- *(Step or milestone)*

## Tasks
- [ ] *(Task to complete)*
- [x] *(Completed task)*

## Useful Information
- *(Insight, gotcha, or reference)*

## Decisions
- `YYYY-MM-DD`: *(Decision and brief rationale)*
```
