#!/bin/bash

# SQLite Memory System - Bash Fallback Script
# Pure bash implementation with sqlite3 CLI and jq only
# Silent failures (exit 0 always)

set -e

# Configuration
DB_PATH="$HOME/.claude/memory.db"
MODE="${1:-load}"

# Helper function to extract ticket from git branch
extract_ticket() {
    local branch
    branch=$(git branch --show-current 2>/dev/null || echo "")
    if [[ -n "$branch" ]]; then
        echo "$branch" | grep -oE '[A-Z]+-[0-9]+' || echo ""
    fi
}

# Initialize database with required schema
init_database() {
    if [[ ! -f "$DB_PATH" ]]; then
        mkdir -p "$(dirname "$DB_PATH")"
        sqlite3 "$DB_PATH" <<'EOF'
CREATE TABLE IF NOT EXISTS memory_sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket TEXT NOT NULL,
    session_data TEXT NOT NULL,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_memory_sessions_ticket ON memory_sessions(ticket);
CREATE INDEX IF NOT EXISTS idx_memory_sessions_timestamp ON memory_sessions(timestamp);
EOF
    fi
}

# Load mode: Show last 5 sessions for ticket
load_memory() {
    echo "🔄 [Claude Memory Hook] Loading context for session..."
    local ticket
    ticket=$(extract_ticket)

    if [[ -z "$ticket" ]]; then
        echo "⚠️  [Claude Memory Hook] No ticket found in branch name"        # No ticket found, return empty array
        echo "[]"
        return 0
    fi

    echo "📋 [Claude Memory Hook] Loading context for ticket: $ticket"
    init_database

    # Query last 5 sessions for the ticket, ordered by timestamp DESC
    local result
    result=$(sqlite3 "$DB_PATH" -json <<EOF
SELECT
    id,
    ticket,
    session_data,
    datetime(timestamp, 'localtime') as timestamp
FROM memory_sessions
WHERE ticket = '$ticket'
ORDER BY timestamp DESC
LIMIT 5;
EOF
)

    # If result is empty, return empty array
    if [[ -z "$result" || "$result" == "[]" ]]; then
        echo "[]"
    else
        echo "$result"
    fi
}

# Save mode: Read JSON from stdin, save to database
save_memory() {
    echo "💾 [Claude Memory Hook] Saving session context..."
    local ticket
    ticket=$(extract_ticket)

    if [[ -z "$ticket" ]]; then
        echo "⚠️  [Claude Memory Hook] No ticket found in branch name"        # No ticket found, exit silently
        return 0
    fi

    init_database

    # Read JSON from stdin
    local session_data
    session_data=$(cat)

    # Validate and compact JSON using jq (also handles escaping)
    local escaped_data
    escaped_data=$(echo "$session_data" | jq -c . 2>/dev/null)
    if [[ -z "$escaped_data" ]]; then
        # Invalid JSON, exit silently
        return 0
    fi

    # Escape single quotes for SQLite
    escaped_data=$(echo "$escaped_data" | sed "s/'/''/g")

    # Insert into database
    sqlite3 "$DB_PATH" <<EOF
INSERT INTO memory_sessions (ticket, session_data)
VALUES ('$ticket', '$escaped_data');
EOF

    echo "✅ [Claude Memory Hook] Context saved for ticket: $ticket"}

# Main execution with error handling
main() {
    case "$MODE" in
        "load")
            load_memory
            ;;
        "save")
            save_memory
            ;;
        *)
            # Unknown mode, exit silently
            exit 0
            ;;
    esac
}

# Execute with silent failure handling
{
    # Execute main function
{
    main "$@"
} || {
    # Silent failure - always exit 0
    exit 0
}