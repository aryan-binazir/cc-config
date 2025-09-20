---
name: Context Sync
description: Keep a single CONTEXT.md in the current dir up to date. Create if missing.
argument-hint: [optional notes or bullets to add to Done/Next]
allowed-tools: Read, Edit, Write, Bash(*)
---

think You maintain a single Markdown "context" file in the CURRENT working directory.
If one exists, update it; if none exists, create CONTEXT.md.

Goals
- Ensure exactly one context Markdown file in the current directory (filename must
  include "context", case-insensitive).
- Normalize structure and keep it current: Plan, Needed Context, Next Actions,
  Done Recently, Risks/Decisions, Archive.
- Add new info from the current session and $ARGUMENTS; prune or archive stale items.

Behavior
1) Detect candidate files in the current dir (non-recursive):
   - Use Bash to list: ls -1
   - Filter case-insensitive names containing "context" and ending in .md or .markdown.
   - If multiple candidates:
     - Prefer (in order): CONTEXT.md, context.md, project-context.md.
     - If still ambiguous, ask me to choose; otherwise auto-pick the most recently
       modified by inspecting ls -lt output.

2) Target file:
   - If none found, set target to CONTEXT.md (strong default).
   - Create it with the template below if it doesn’t exist.

3) Normalize structure in the target file:
   - Maintain a single managed block delimited by:
     <!-- context:managed:start -->
     ...
     <!-- context:managed:end -->
   - Only rewrite inside the managed block. Preserve any content outside of it.

4) Compose updates from:
   - This chat’s current plan and discussion.
   - $ARGUMENTS (treat free text as bullets: if it starts with "done:" put in Done
     Recently; if "next:" put in Next Actions; otherwise infer).
   - If a git repo, optionally enrich Done Recently with the latest commit subjects
     (git log -n 5 --pretty=format:"- %s") and current changes (git status --porcelain)
     when helpful.

5) De-duplicate and prune:
   - Move items completed more than ~14 days ago from Done Recently into Archive.
   - Remove obsolete or superseded items; if unsure, ask me.
   - Keep lists concise and actionable.

6) Always update the "Last updated" timestamp in the header with local time.

7) Show me a brief summary of changes (or a diff if practical), then save edits.

Template (for new files or when adding the managed block)
- If creating, write this whole file; if updating, ensure the managed block matches.

------------------------------------------------------------------------------
# Project Context

Last updated: {{local_time_here}}

<!-- context:managed:start -->

## Plan (Current)
- High-level plan for the workstream with 3–7 bullets max.

## Needed Context / Open Questions
- Unknowns, blockers, and info we must obtain.

## Next Actions
- Short, imperative, verifiable tasks.
- Each item should be small enough to complete in one sitting.

## Done Recently
- What we’ve completed lately (last ~2 weeks).

## Risks / Decisions
- Notable risks, tradeoffs, and decisions with dates.

## Archive (auto)
- Older done items and retired notes live here.

<!-- context:managed:end -->

Notes
- Keep sections terse. Prefer clarity over verbosity.
- Do not create extra context files. Always converge on a single target file.
- If Windows shell lacks common Unix tools, use built-in file listing or Git Bash/WSL.

Execution steps (concrete)
- List candidates:
  - Bash: ls -1 | grep -i "context" | grep -Ei "\.(md|markdown)$" || true
- Choose target as per rules above; ask if ambiguous.
- If creating: write the full template with current timestamp.
- If updating:
  - Read the existing file.
  - Parse or insert the managed block.
  - Synthesize updated sections using session content and $ARGUMENTS.
  - Move stale Done items to Archive; drop obvious cruft.
  - Update Last updated.
- Present a short summary of changes; then save.
