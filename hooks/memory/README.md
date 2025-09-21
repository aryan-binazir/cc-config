# Claude Code Memory Hooks System

An intelligent memory system for Claude Code that automatically captures and categorizes work context by git branch/ticket. Stores up to 50 categorized context points including code patterns, decisions, implementations, and TODOs.

## Features

- **Automatic context capture** from your Claude conversations
- **Smart categorization** into 5 types: decisions, implementations, code patterns, state, next steps
- **Code extraction** from messages and git diffs
- **50-point capacity** intelligently distributed across categories
- **Session tracking** with file modifications and task descriptions
- **Ticket-based organization** from git branch names (JIRA-123, PROJ-456)
- **Dual implementation** - Go binary with bash fallback

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
- jq (optional, for bash fallback)

## How It Works

Hybrid approach combining:
- **SessionStart Hook** - Auto-loads context and creates/syncs CONTEXT.md
- **Slash Commands** - Manual capture after Claude works (since hooks can't see Claude's output)
- **Git Integration** - Extracts actual code changes from git diff
- **CONTEXT.md** - Visible context file that syncs with database

## Context Categories

The system stores up to 50 context points across 5 categories:

### 💡 Decisions (10 max)
Strategic choices with reasoning:
- "Use Redis because it handles horizontal scaling"
- "Chose Paseto over JWT for better security"

### 🏗️ Implementations (15 max)
What's been built:
- "POST /api/login - Returns JWT token"
- "Added rate limiting middleware"

### 🔧 Code Patterns (15 max)
Function signatures and types extracted from code:
- `func AuthRequired(next http.HandlerFunc) http.HandlerFunc`
- `type Token struct { UserID string; Exp int64 }`

### 📊 Current State (10 max)
What works/what's broken:
- "✅ Login endpoint working with rate limiting"
- "❌ Refresh tokens not implemented"

### 📝 Next Steps (10 max)
TODOs and blockers:
- "TODO: Implement refresh tokens"
- "BLOCKED: Need Redis credentials from DevOps"

## Slash Commands (Primary Interface)

```bash
# After implementing something
/memory_sync                    # Captures git diff + syncs CONTEXT.md

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

# Sync with CONTEXT.md
~/.claude/hooks/memory/memory context sync-context

# Manual saves
~/.claude/hooks/memory/memory context save decision PROJ-456 "Use Redis for scaling"
~/.claude/hooks/memory/memory context save implementation PROJ-456 "Added auth middleware"
~/.claude/hooks/memory/memory context save pattern PROJ-456 "func ValidateToken(token string)"
~/.claude/hooks/memory/memory context save state PROJ-456 "✅ Tests passing"
~/.claude/hooks/memory/memory context save next PROJ-456 "TODO: Add rate limiting"
```

## CONTEXT.md Integration

The system maintains a CONTEXT.md file in your working directory:
- Auto-created on session start if missing
- Syncs bidirectionally with database
- Visible to you and Claude during work
- Standard format compatible with /context_sync command

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

1. **Session starts**: Context auto-loads, CONTEXT.md created/synced
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
- Use CONTEXT.md for visibility

## Database Schema

### ticket_context_enhanced
Stores categorized context with up to 50 points total:
```sql
CREATE TABLE ticket_context_enhanced (
    ticket TEXT PRIMARY KEY,
    requirements TEXT,
    decisions TEXT,          -- JSON array (max 10)
    implementations TEXT,    -- JSON array (max 15)
    code_patterns TEXT,      -- JSON array (max 15)
    current_state TEXT,      -- JSON array (max 10)
    next_steps TEXT,         -- JSON array (max 10)
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

**CONTEXT.md not syncing:**
- File must be in current working directory
- Must have managed block markers (<!-- context:managed:start/end -->)
- Run `/memory_sync` to force sync

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