# Claude Code Memory Hooks System

An intelligent memory system for Claude Code that automatically captures and categorizes work context by git branch/ticket. Stores up to 100 categorized context points including code patterns, decisions, implementations, and TODOs.

## Features

- **Automatic context capture** from your Claude conversations
- **Smart categorization** into 5 types: decisions, implementations, code patterns, state, next steps
- **Code extraction** from messages and git diffs
- **100-point capacity** intelligently distributed across categories
- **Session tracking** with git modifications and task descriptions
- **Ticket-based organization** from git branch names (JIRA-123, PROJ-456)

## Quick Installation

```bash
git clone https://github.com/yourusername/cc-memory-hooks.git
cd cc-memory-hooks
./install.sh
```

Installs to `~/.claude/hooks/memory/` and configures minimal hooks (SessionStart only).

## Requirements

- Go 1.25+ (for building)
- SQLite3

## How It Works

The memory system uses:
- **SessionStart Hook** - Auto-loads context from SQLite database
- **Slash Commands** - Manual capture after Claude works (since hooks can't see Claude's output)
- **Git Integration** - Extracts actual code changes from git diff
- **SQLite Database** - Persistent storage for all context data

## Context Categories

The system stores up to 100 context points across 5 categories:

### 💡 Decisions (20 max)
Strategic choices with reasoning:
- "Use Redis because it handles horizontal scaling"
- "Chose Paseto over JWT for better security"

### 🏗️ Implementations (50 max)
What's been built:
- "POST /api/login - Returns JWT token"
- "Added rate limiting middleware"

### 🔧 Code Patterns (30 max)
Function signatures and types extracted from code:
- `func AuthRequired(next http.HandlerFunc) http.HandlerFunc`
- `type Token struct { UserID string; Exp int64 }`

### 📊 Current State (20 max)
What works/what's broken:
- "✅ Login endpoint working with rate limiting"
- "❌ Refresh tokens not implemented"

### 📝 Next Steps (20 max)
TODOs and blockers:
- "TODO: Implement refresh tokens"
- "BLOCKED: Need Redis credentials from DevOps"

## Slash Commands (Primary Interface)

```bash
# After implementing something
/memory_sync                    # Captures git diff patterns

# Save specific context
/memory_decision [text]         # Save architectural decision
/memory_implementation [text]   # Save what was built
/memory_todo [text]            # Save TODO or blocker

# Review
/memory_review                 # Show all context for current ticket
```

## Direct Commands

```bash
# Sync with git diff
~/.claude/hooks/memory/memory context sync-git


# Manual saves
~/.claude/hooks/memory/memory context save decision PROJ-456 "Use Redis for scaling"
~/.claude/hooks/memory/memory context save implementation PROJ-456 "Added auth middleware"
~/.claude/hooks/memory/memory context save pattern PROJ-456 "func ValidateToken(token string)"
~/.claude/hooks/memory/memory context save state PROJ-456 "✅ Tests passing"
~/.claude/hooks/memory/memory context save next PROJ-456 "TODO: Add rate limiting"
```

## SQLite-Only Storage

The system stores all context data in a SQLite database:
- Located at `~/.claude/memory.db`
- No file dependencies or syncing required
- Fast queries and updates
- Automatic categorization and limits

## Database Cleanup

```bash
# Remove sessions older than 30 days (default)
~/.claude/hooks/memory/memory cleanup

# Remove sessions older than 180 days
~/.claude/hooks/memory/memory cleanup 180
```

### Query Tool

```bash
# Find blockers across all tickets
~/.claude/hooks/memory/query blockers

# Show all TODOs
~/.claude/hooks/memory/query todos

# Show technical decisions
~/.claude/hooks/memory/query decisions

# Show all tickets with summary
~/.claude/hooks/memory/query all

# Show recent updates
~/.claude/hooks/memory/query recent
```

## Example Context Display

```
📝 Recent work on PROJ-456:
  • 2025-09-20 17:34 (1m): Implement auth system with JWT
  • 2025-09-20 17:32 (1m): Modified: auth.go, middleware.go

📊 Total: 5 sessions, 45 minutes

╔════════════════════════════════════════════════════════════
║ 🎯 CONTEXT FOR PROJ-456
╚════════════════════════════════════════════════════════════

📋 REQUIREMENTS:
Build JWT-based auth with 2FA support, 15-minute token expiry

💡 KEY DECISIONS (3):
  • 📌 Use Paseto instead of JWT for security
  • Redis for sessions due to horizontal scaling
  • bcrypt cost 12 for password hashing

🏗️ IMPLEMENTATIONS (5):
  • POST /api/login - Returns auth token
  • POST /api/logout - Invalidates session
  • GET /api/user - Returns current user
  • Auth middleware for /api/* routes
  • Rate limiting: 5 attempts/min

🔧 CODE PATTERNS (8):
  • func AuthRequired(next http.HandlerFunc) http.HandlerFunc
  • type Token struct { UserID string; Exp int64 }
  • func ValidateToken(token string) error
  • func HashPassword(password string) string
  • router.HandleFunc("/api/*", AuthRequired(handler))
  • func CreateUser(w http.ResponseWriter, r *http.Request)
  • type User struct { ID string; Email string; Role string }
  • func GetRedisClient() *redis.Client

📊 CURRENT STATE (3):
  • ✅ Login/logout working
  • ✅ Session storage in Redis
  • ❌ Refresh tokens not implemented

📝 NEXT STEPS / TODOs (2):
  • TODO: Implement refresh tokens for mobile
  • BLOCKED: Need Redis prod credentials

═══════════════════════════════════════════════════════════
```

## The Honest Workflow

1. **Session starts**: Context auto-loads from database
2. **Work with Claude**: Makes implementations
3. **After implementing**: Run `/memory_sync` to capture git diff
4. **After decisions**: Run `/memory_decision [reasoning]`
5. **Review anytime**: Run `/memory_review`

## Git Diff Extraction

The `/memory_sync` command captures from git diff:
- Function signatures: `func FunctionName`
- Type definitions: `type TypeName`
- Method signatures: `method MethodName`
- Interface definitions: `interface InterfaceName`

## Why This Architecture?

**The Problem**: Claude Code hooks only provide ${LAST_HUMAN_MESSAGE}, not Claude's responses or code

**The Solution**:
- Use hooks minimally (just to load context)
- Use slash commands for manual capture after work
- Use git diff to capture actual code changes
- Use SQLite for fast, reliable storage

## Database Schema

### ticket_context_enhanced
Stores categorized context with up to 100 points total:
```sql
CREATE TABLE ticket_context_enhanced (
    ticket TEXT PRIMARY KEY,
    requirements TEXT,
    decisions TEXT,          -- JSON array (max 20)
    implementations TEXT,    -- JSON array (max 50)
    code_patterns TEXT,      -- JSON array (max 30)
    current_state TEXT,      -- JSON array (max 20)
    next_steps TEXT,         -- JSON array (max 20)
    created_at DATETIME,
    last_updated DATETIME
);
```

### sessions
Tracks work sessions per ticket:
```sql
CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    ticket TEXT,
    session_id TEXT UNIQUE,
    task_description TEXT,
    files_modified TEXT,     -- JSON array
    lines_added INTEGER,
    lines_removed INTEGER,
    start_time DATETIME,
    end_time DATETIME,
    commit_sha TEXT
);
```

## Configuration

Minimal hook configuration in `~/.claude/settings.json`:

```json
{
  "hooks": {
    "SessionStart": [{
      "hooks": [{
        "type": "command",
        "command": "$HOME/.claude/hooks/memory/memory load"
      }]
    }]
  }
}
```

Note: We only use SessionStart because hooks can't capture Claude's actual work (no ${LAST_ASSISTANT_MESSAGE})

## Troubleshooting

**Context not loading at session start:**
- Check that SessionStart hook is configured in settings.json
- Verify branch has a ticket pattern (JIRA-123, PROJ-456)

**No automatic capture:**
- This is by design - hooks can't see Claude's work
- Use `/memory_sync` after Claude implements features
- Use `/memory_decision` etc for specific captures

**Context not saving:**
- Check that database is writable at `~/.claude/memory.db`
- Verify branch has a ticket pattern (JIRA-123, PROJ-456)
- Use debug mode: `CLAUDE_MEMORY_DEBUG=1` to see detailed logs

**Database location:**
- Database is at `~/.claude/memory.db`
- Use SQLite browser to inspect directly

## Performance

- Load time: ~20ms
- Save time: ~40ms
- Database size: ~5KB per ticket with full context
- Memory usage: <15MB

## License

MIT