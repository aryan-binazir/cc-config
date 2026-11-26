# cc-config

A common place for skills and other shareable agent configuration used across
Codex, Claude, Cursor, and related tools.

`AGENTS.md` is the canonical shared instruction file. Tool-specific files such
as `CLAUDE.md` can symlink back to it when needed.

## Hooks

[`agent-hooks/`](agent-hooks/README.md) is one Go binary that enforces agent
guardrails as `PreToolUse`/`PostToolUse` hooks in Claude Code, Codex, and the
Antigravity CLI. It blocks agents from launching agy, and it enforces draft
and assigned PRs, safe merges, commit-message and PR-title format, and no
secrets, production pushes, or destructive git. `make -C agent-hooks install` builds it
to `~/.local/bin/agent-hooks` and wires it into `~/.claude/settings.json`,
`~/.codex/hooks.json`, and `~/.gemini/config/hooks.json`. Codex then needs the
new entries approved in `/hooks`.

`hooks/block-agy.pi.ts` is a pi extension that runs the block-agy check through
`agent-hooks pre --only=block-agy`.
