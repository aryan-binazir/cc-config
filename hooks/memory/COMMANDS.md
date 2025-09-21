# Enhanced Memory System Commands

The enhanced memory system now supports categorized context with up to 50 points across five categories:
- **Decisions** (max 10): Architectural and design choices
- **Implementations** (max 15): What's been built
- **Code Patterns** (max 15): Function signatures and types
- **Current State** (max 10): What works/what's broken
- **Next Steps** (max 10): TODOs and blockers

## Category-Specific Save Commands

### Save a Decision
```bash
memory context save decision <ticket> "Using Paseto instead of JWT because of security audit"
# Or auto-detect ticket from branch:
memory context save decision "Using Redis for session storage due to horizontal scaling"
```

### Save an Implementation
```bash
memory context save implementation <ticket> "POST /api/login - Returns Paseto token with 15min expiry"
# Or auto-detect:
memory context save implementation "Added rate limiting middleware (5 attempts/min)"
```

### Save a Code Pattern
```bash
memory context save pattern <ticket> "func AuthRequired(next http.HandlerFunc) http.HandlerFunc"
# Or auto-detect:
memory context save pattern "type Token struct { UserID string; Exp int64 }"
```

### Save Current State
```bash
memory context save state <ticket> "✅ Login/logout working with Paseto tokens"
memory context save state <ticket> "❌ Refresh tokens not implemented"
memory context save state <ticket> "⚠️ Breaks when Redis disconnects"
```

### Save Next Steps
```bash
memory context save next <ticket> "TODO: Implement refresh tokens for mobile app"
memory context save next <ticket> "BLOCKED: Need Redis prod credentials from DevOps"
```

## Automatic Categorization

If you don't specify a category, the system will auto-categorize based on content:

```bash
memory context add <ticket> "func ValidateToken(tokenString string) (*Token, error)"
# Auto-categorized as: pattern

memory context add <ticket> "TODO: Add rate limiting"
# Auto-categorized as: next

memory context add <ticket> "Using bcrypt cost 10 because of performance constraints"
# Auto-categorized as: decision
```

## Automatic Code Pattern Extraction

The system automatically extracts code patterns from git diffs when you save a session:
- Function signatures: `func AuthRequired(next http.HandlerFunc) http.HandlerFunc`
- Type definitions: `type Session struct { ID string; UserID string }`
- API endpoints: `router.HandleFunc("/api/login", LoginHandler)`
- Router patterns: `router.Get("/api/user", GetUser)`

These are automatically saved to the "Code Patterns" category.

## Loading Enhanced Context

```bash
memory context load <ticket>
# Or auto-detect from branch:
memory context load
```

This displays categorized context in an enhanced format:

```
╔════════════════════════════════════════════════════════════
║ 🎯 PROJ-123
╚════════════════════════════════════════════════════════════

📋 REQUIREMENTS:
Build a secure authentication system with JWT tokens

🏗️ IMPLEMENTATIONS (3):
  • POST /api/login - Returns Paseto token
  • POST /api/logout - Invalidates session in Redis
  • GET /api/user - Returns current user (requires auth)

💡 KEY DECISIONS (2):
  • Using Paseto instead of JWT because of security audit
  • 15-minute session timeout for compliance

🔧 CODE PATTERNS (3):
  • func AuthRequired(next http.HandlerFunc) http.HandlerFunc
  • type Token struct { UserID string; Exp int64 }
  • router.HandleFunc("/api/*", AuthRequired(handler))

📊 CURRENT STATE (2):
  • ✅ Login/logout working with Paseto tokens
  • ❌ Rate limiting not implemented

📝 NEXT STEPS (2):
  • TODO: Implement refresh tokens
  • BLOCKED: Need Redis credentials

═══════════════════════════════════════════════════════════
```

## Listing Tickets with Enhanced Context

```bash
memory context list
```

Shows all tickets with context counts by category:
```
• PROJ-123 (25 total: 💡5 🏗️8 🔧6 📊3 📝3) - Updated: 2025-09-20 15:04
  Requirements: Build secure authentication system...
• PROJ-456 (12 total: 💡2 🏗️4 🔧3 📊2 📝1) - Updated: 2025-09-19 10:30
```

## Requirements

Set high-level requirements for a ticket:
```bash
memory context requirements <ticket> "Build a GraphQL API with real-time subscriptions and authentication"
```

## Clear Context

Clear all context for a ticket (requires confirmation):
```bash
memory context clear <ticket>
```

## Integration with Claude Code

When using Claude Code, you can save context directly from your conversation using slash commands:

**IMPORTANT NOTE**: Memory commands are for **documentation and tracking purposes only**.
They do NOT automatically execute tasks or implement decisions. They simply record
information to your project context for future reference and planning.

### Slash Commands Behavior:
- `/memory_todo` - Records a TODO to context, does NOT start working on it
- `/memory_implementation` - Documents what was/will be implemented, does NOT execute code
- `/memory_decision` - Records architectural decisions, does NOT implement them
- `/memory_sync` - Captures current work state for documentation
- `/memory_review` - Displays saved context for review

### Examples:
1. **For decisions**: "Remember: Using Paseto instead of JWT for better security"
2. **For TODOs**: "TODO: Add refresh token endpoint" (recorded but not executed)
3. **For blockers**: "BLOCKED: Waiting on DevOps for Redis credentials"
4. **For state**: "✅ Authentication is now working"

The system will automatically categorize and save these based on keywords and patterns,
but will NOT automatically start working on them unless you explicitly request it.

## Tips for Effective Context Management

1. **Be Specific with Code Patterns**: Include full function signatures
   ```bash
   memory context save pattern "func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request)"
   ```

2. **Use Status Emojis for State**: Makes scanning easier
   - ✅ for working features
   - ❌ for broken/missing features
   - ⚠️ for warnings/issues
   - 🚧 for work in progress

3. **Include "because" in Decisions**: Helps Claude understand reasoning
   ```bash
   memory context save decision "Using PostgreSQL instead of MySQL because of better JSON support"
   ```

4. **Be Specific with Implementations**: Include endpoint methods
   ```bash
   memory context save implementation "POST /api/v1/users - Creates new user with email validation"
   ```

5. **Link Blockers to External Systems**: Include ticket numbers
   ```bash
   memory context save next "BLOCKED: Waiting on INFRA-789 for database credentials"
   ```