# cc-config

A common place for skills and other shareable agent configuration used across
Codex, Claude, Cursor, and related tools.

`AGENTS.md` is the canonical shared instruction file. Tool-specific files such
as `CLAUDE.md` can symlink back to it when needed.

## Hooks

`hooks/block-agy.sh` stops agents from launching the Antigravity CLI in any
mode (print, ACP, or driven through tmux), which Google's consumer-plan terms
forbid. Running agy yourself in a terminal is unaffected. It is a
`PreToolUse` hook for Claude Code (`~/.claude/settings.json`) and Codex
(`~/.codex/hooks.json`), and `hooks/block-agy.pi.ts` runs it in pi.
Both files are symlinked into place:

- `~/.claude/hooks/block-agy.sh` -> `hooks/block-agy.sh`
- `~/.pi/agent/extensions/block-agy.ts` -> `hooks/block-agy.pi.ts`
