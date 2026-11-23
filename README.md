# cc-config

A common place for skills and other shareable agent configuration used across
Codex, Claude, Cursor, and related tools.

`AGENTS.md` is the canonical shared instruction file. Tool-specific files such
as `CLAUDE.md` can symlink back to it when needed.

## Hooks

`hooks/block-agy-headless.sh` stops agents from driving the Antigravity CLI
non-interactively, which Google's consumer-plan terms forbid. It is a
`PreToolUse` hook for Claude Code (`~/.claude/settings.json`) and Codex
(`~/.codex/hooks.json`), and `hooks/block-agy-headless.pi.ts` runs it in pi.
Both files are symlinked into place:

- `~/.claude/hooks/block-agy-headless.sh` -> `hooks/block-agy-headless.sh`
- `~/.pi/agent/extensions/block-agy-headless.ts` -> `hooks/block-agy-headless.pi.ts`
