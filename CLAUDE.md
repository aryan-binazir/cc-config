# Communication
- Call me Ar.
- Be direct, no fluff. Give brutally honest pushback when I'm wrong.
- Ask questions during planning. During execution, make reasonable assumptions and state them.
- If a decision is hard to undo or changes user-facing behavior/scope, stop and ask.

# Workflow
- _scratch/_context/{branch}.md is the source of truth for project state.
- Update context when plans, assumptions, or decisions change.
- Before major work, check for project-specific agent rules (CLAUDE.md, .cursorrules, AGENTS.md, etc).

# Self-Improvement Loop
- After ANY correction from the user: if the current workspace is inside a git repo, update ~/_scratch/_lessons/lessons.md with the pattern. If it is not inside a git repo, do not create _scratch.
- Write rules for yourself that prevent the same mistake.
- Ruthlessly iterate on these lessons until mistake rate drops.
- Review lessons at session start for relevant project.

# Git Rules
- Do not force-push or rewrite history.
- Never add Co-Authored-By lines to commits.
- Commit message format: `type(TICKET): description` (e.g. `fix(BBA-9): Added logging package`). Types: `feat`, `fix`, `chore`, `refactor`.
  - The type is determined by the ticket's overall purpose and MUST stay consistent across ALL commits and the PR title for that ticket. Check existing commits on the branch before committing.
- PR descriptions use this structure. Testing is optional — omit if not applicable:
  ### Problem
  ### Changes
  ### Decisions
  ### Testing
  How it was tested, or how to test it.
  Followed by a collapsed `<details>` block titled "Agent Context" containing generous, verbose context a future agent reviewing this PR would want — original ask, why this approach, constraints, gotchas, anything non-obvious. Err on the side of too much rather than too little.
