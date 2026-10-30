# Communication
- Call me Ar.
- Be direct, skip the fluff, and push back bluntly you believe I'm wrong.
- Make responses as concise as possible without losing meaning.
- Ask questions during planning. During execution, make reasonable assumptions and state them.
- If a decision is a one-way door, i.e. going through it would be exceptionally hard to back out of, or changes user-facing behavior/scope, stop and ask. Two-way doors present far less risk to merge, since they can easily be reverted.

# Workflow
- I work tickets in worktrees, keep ticket/worktree context in `_scratch/CONTEXT.md`.
- Skip context files entirely for one-off explorations, quick scripts, and questions, or if we aren't in a worktree.
- Keep the active context file current when plans, assumptions, or decisions change; delete stale notes rather than accumulating them.

# Engineering
- Done means verified: run the project's typecheck/lint and relevant tests before declaring work complete. Report failures honestly.
- Prefer the simplest solution that works. If you see a simpler approach than what I asked for, propose it.
- Consider whether deleting code and reworking abstractions in addition to the feature work is preferable than simply bolting a feature onto the side of the existing features.
- Prefer few deep modules over many shallow ones: simple interfaces hiding complex implementations. If a wrapper, helper, or layer barely simplifies what it wraps, inline it.
- Don't extract code just to make it testable, and don't add an abstraction for a single caller — test through the module's real interface, and add the layer when a second consumer actually exists.

## Git conventions
- Branches for Jira-backed work should be named `amb/XXXX-XXX`, where `XXXX-XXX` is the Jira issue key.
- Commit message format: `type(PROJECT): description` (e.g. `fix(BBA): Added logging package`). Use only the Jira project key, not the full ticket key. Types: `feat`, `fix`, `chore`, `refactor`, `test`, `docs`, `build`, `ci`, `revert`. If there is no ticket, use `type(no-ticket): description`.
  - The type is determined by the ticket's overall purpose and MUST stay consistent across ALL commits and the PR title for that ticket (CI depends on this). A stack is one larger change split for review, so a `refactor` commit inside a `feat` ticket is still `feat`. Check existing commits on the branch before committing.
- Use the pr skill when creating pull requests.
- If ticketed, end the PR body with `<Type> <ticket ID>` (`Fixes` for `fix`, otherwise capitalize the commit type).
- PR descriptions use this structure. Testing is optional — omit if not applicable:
  ### Problem
  ### Changes
  ### Decisions
  ### Testing
  How it was tested, or how to test it.
  Follow this with a collapsed `<details>` block titled "Agent Context" containing any detailed context an agent would find helpful when reviewing the PR. Make this block as thorough/verbose as needed to help future agents understand our context.
- When opening a PR, always make it a draft, and assign the PR to Aryan Binazir.
