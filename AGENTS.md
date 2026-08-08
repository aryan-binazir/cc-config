# Communication
- Call me Ar.
- Be direct, skip the fluff, and push back bluntly when I'm wrong.
- Make responses as concise as possible without losing meaning.
- Ask questions during planning. During execution, make reasonable assumptions and state them.
- If a decision is hard to undo or changes user-facing behavior/scope, stop and ask.

# Workflow
- For ticket- or branch-backed work, project state lives in `_scratch/_context/<ticket-key>.md`. Resolve the key from the ticket or intended branch, not from whatever branch is checked out; ask me if you can't determine it. Only use this for explicit ticket/feature branches — never `_scratch/_context/main.md`.
- Skip context files entirely for one-off explorations, quick scripts, and questions.
- Keep the active context file current when plans, assumptions, or decisions change; delete stale notes rather than accumulating them.

# Engineering
- Don't start dev servers or run build commands unless asked — assume dev servers are already running. Verify with check commands (typecheck, lint, tests) instead.
- Done means verified: run the project's typecheck/lint and relevant tests before declaring work complete. Report failures honestly.
- Prefer the simplest solution that works. If you see a simpler approach than what I asked for, propose it.
- Prefer few deep modules over many shallow ones: simple interfaces hiding complex implementations. If a wrapper, helper, or layer barely simplifies what it wraps, inline it.
- Don't extract code just to make it testable, and don't add an abstraction for a single caller — test through the module's real interface, and add the layer when a second consumer actually exists.

# Version Control

- Use `git` directly. Prefer a separate Git worktree for isolated work.
- Prefer a stack of small changes over one large change or PR: each change has one clear purpose, is independently reviewable, and is created explicitly in stack order.
- Keep stacked pull requests in draft until the full stack is reviewed and stable. Before publishing, inspect the stack, verify parent relationships, run relevant tests, and ensure each change has a clear description.
- Do not rewrite, squash, reorder, or abandon pushed changes without explaining the impact first.
- When uncertain about repository state, stop and show `git status` and `git log --oneline --decorate --graph -20` rather than attempting speculative recovery.

## Git conventions
- Branches for Jira-backed work should be named `aryan-binazir/XXXX-XXX`, where `XXXX-XXX` is the Jira issue key.
- Commit message format: `type(PROJECT): description` (e.g. `fix(BBA): Added logging package`). Use only the Jira project key, not the full ticket key. Types: `feat`, `fix`, `chore`, `refactor`. If there is no ticket, use `type(no-ticket): description` — never invent a plausible-looking ticket key.
  - The type is determined by the ticket's overall purpose and MUST stay consistent across ALL commits and the PR title for that ticket (CI depends on this). A stack is one larger change split for review, so a `refactor` commit inside a `feat` ticket is still `feat`. Check existing commits on the branch before committing.
- PR descriptions use this structure. Testing is optional — omit if not applicable:
  ### Problem
  ### Changes
  ### Decisions
  ### Testing
  How it was tested, or how to test it.
  Follow this with a collapsed `<details>` block titled "Agent Context" containing any detailed context an agent would find helpful when reviewing the PR. Make this block as thorough/verbose as needed to help future agents understand our context.
- When opening a PR, assign the PR to Aryan Binazir.
