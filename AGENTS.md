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
- When a skill under ~/repos/cc-config/skills causes wasted effort, confusion, or a wrong turn, the agent may ask permission to improve that skill. Keep this limited to skill/process improvements; do not silently expand the user's current task.
- If the user grants permission and the agent changes a skill, the final response must say which skill changed, summarize what changed, and offer a concrete revert or adjustment path.

# Git Rules
- Branches for Jira-backed work should be named `aryan-binazir/XXXX-XXX`, where `XXXX-XXX` is the Jira issue key.
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
