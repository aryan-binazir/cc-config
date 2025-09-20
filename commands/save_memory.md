# Create /save_memory Slash Command

## File: `.claude/commands/save_memory.md`

```markdown
Save this to the SQLite memory database for the current ticket:

$ARGUMENTS

Run this command to add it to the ticket context:
`$CLAUDE_PROJECT_DIR/.claude/hooks/memory/memory context add "$ARGUMENTS"`

This adds a permanent context point to the SQLite database at ~/.claude/memory.db in the ticket_context table.

If the argument starts with "requirements:" or "r:", save it as requirements instead of a context point.

Confirm what was saved and to which ticket.
```

## That's it.

The slash command just calls your existing `memory` binary that already:
- Detects the current git branch/ticket
- Connects to the SQLite database at `~/.claude/memory.db`
- Adds to the `ticket_context` table
- Handles requirements vs context points

## Usage

```bash
/save_memory Redis creds are in 1Password
/save_memory requirements: Build auth with JWT and 2FA  
/save_memory Always validate tokens before trusting claims
```

The memory tool does all the work. The slash command just passes your text to it.
