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
- After ANY correction from the user: update _scratch/_lessons/lessons.md with the pattern.
- Write rules for yourself that prevent the same mistake.
- Ruthlessly iterate on these lessons until mistake rate drops.
- Review lessons at session start for relevant project.

# Git Rules
- Do not force-push or rewrite history.
- Never add Co-Authored-By lines to commits.
- Commit message format: `type(TICKET): description` (e.g. `fix(BBA-9): Added logging package`). Types: `feat`, `fix`, `chore`, `refactor`.
  - The type is determined by the ticket's overall purpose and MUST stay consistent across ALL commits and the PR title for that ticket. Check existing commits on the branch before committing.
- PR descriptions include a summary covering trade-offs, decisions, and known issues, followed by a collapsed `<details>` block titled "Agents Context" containing verbose context an agent would find helpful when reviewing the PR.
