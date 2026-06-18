# Communication
- Call me Ar.
- Be direct, no fluff. Give brutally honest pushback when I'm wrong.
- Ask questions during planning. During execution, make reasonable assumptions and state them.
- If a decision is hard to undo or changes user-facing behavior/scope, stop and ask.

# Workflow
- _scratch/_context/{branch}.md is the source of truth for project state.
- Update context when plans, assumptions, or decisions change.
- Before major work, check for project-specific agent rules (AGENTS.md, CLAUDE.md, .cursorrules, etc).

# Self-Improvement Loop
- After any correction from the user, update ~/_scratch/_lessons/lessons.md if you are not in a git repo, or _scratch/_lessons/lessons.md if the repo has that pattern.
- Write rules for yourself that prevent the same mistake.
- Ruthlessly iterate on these lessons until mistake rate drops.
- Review lessons at session start for relevant project.

# Git Rules
- Commit message format: `type(TICKET): description` (e.g. `fix(BBA-9): Added logging package`). Types: `feat`, `fix`, `chore`, `refactor`. If the current branch does not include a ticket ID, choose a reasonable ticket name.
  - The type is determined by the ticket's overall purpose and MUST stay consistent across ALL commits and the PR title for that ticket. Check existing commits on the branch before committing.
- PR descriptions use this structure. Testing is optional — omit if not applicable:
  ### Problem
  ### Changes
  ### Decisions
  ### Testing
  How it was tested, or how to test it.
  Follow this with a collapsed `<details>` block titled "Agent Context" containing any detailed context an agent would find helpful when reviewing the PR. Make this block as thorough/verbose as needed to help future agents understand our context.
- When opening a PR, assign the PR to Aryan Binazir.
