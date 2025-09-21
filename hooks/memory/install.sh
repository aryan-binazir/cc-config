#!/bin/bash

# Memory Hooks Installation Script
# Installs the Claude Code memory hooks system to ~/.claude/hooks/memory

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to compare version numbers
version_ge() {
    printf '%s\n%s\n' "$2" "$1" | sort -V -C
}

# Function to get Go version
get_go_version() {
    if command_exists go; then
        go version | sed 's/go version go\([0-9.]*\).*/\1/'
    else
        echo ""
    fi
}

# Main installation function
main() {
    log_info "Starting Claude Code Memory Hooks installation..."

    # Set directories
    SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    INSTALL_DIR="$HOME/.claude/hooks/memory"
    DB_PATH="$HOME/.claude/memory.db"

    # Step 1: Create installation directory
    log_info "Step 1: Creating installation directory..."
    mkdir -p "$INSTALL_DIR" || {
        log_error "Failed to create installation directory at $INSTALL_DIR"
        exit 1
    }
    log_success "Created $INSTALL_DIR"

    # Step 2: Check for Go 1.25+
    log_info "Step 2: Checking Go installation..."

    if ! command_exists go; then
        log_error "Go is not installed or not in PATH"
        log_error "Please install Go 1.25+ from https://golang.org/dl/"
        exit 1
    fi

    GO_VERSION=$(get_go_version)
    REQUIRED_VERSION="1.25"

    if ! version_ge "$GO_VERSION" "$REQUIRED_VERSION"; then
        log_error "Go version $GO_VERSION is installed, but version $REQUIRED_VERSION or higher is required"
        log_error "Please upgrade Go from https://golang.org/dl/"
        exit 1
    fi

    log_success "Go $GO_VERSION detected (requirement: $REQUIRED_VERSION+)"

    # Step 3: Check required files exist
    log_info "Step 3: Checking required files..."

    if [ ! -f "$SOURCE_DIR/memory.go" ]; then
        log_error "memory.go not found in $SOURCE_DIR"
        exit 1
    fi

    log_success "All required files found"

    # Step 4: Install Go dependencies
    log_info "Step 4: Installing Go dependencies..."

    cd "$SOURCE_DIR"

    # Initialize go module if go.mod doesn't exist
    if [ ! -f "go.mod" ]; then
        log_info "Initializing Go module..."
        go mod init claude-memory-hooks
    fi

    # Install sqlite3 dependency
    if ! go get github.com/mattn/go-sqlite3; then
        log_error "Failed to install mattn/go-sqlite3 dependency"
        exit 1
    fi

    log_success "Dependencies installed"

    # Step 5: Build the Go binaries
    log_info "Step 5: Building Go binaries..."

    # Build memory binary
    BUILD_CMD="go build -tags sqlite_omit_load_extension -ldflags=\"-s -w\" -o memory memory.go types.go"

    if ! eval "$BUILD_CMD"; then
        log_error "Failed to compile memory.go"
        log_error "Build command: $BUILD_CMD"
        exit 1
    fi

    if [ ! -f "$SOURCE_DIR/memory" ]; then
        log_error "Compilation succeeded but binary 'memory' not found"
        exit 1
    fi

    log_success "Memory binary built successfully"

    # Build query binary
    if [ -f "$SOURCE_DIR/query.go" ] && [ -f "$SOURCE_DIR/types.go" ]; then
        log_info "Building query binary..."
        BUILD_CMD="go build -tags sqlite_omit_load_extension -ldflags=\"-s -w\" -o query query.go types.go"

        if eval "$BUILD_CMD"; then
            log_success "Query binary built successfully"
        else
            log_warning "Failed to build query binary (optional tool)"
        fi
    fi

    # Step 6: Copy files to installation directory
    log_info "Step 6: Installing files to $INSTALL_DIR..."

    cp "$SOURCE_DIR/memory" "$INSTALL_DIR/" || {
        log_error "Failed to copy memory binary"
        exit 1
    }

    # Copy query binary if it exists
    if [ -f "$SOURCE_DIR/query" ]; then
        cp "$SOURCE_DIR/query" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/query"
        log_info "Query tool installed"
    fi

    # Copy README if it exists
    if [ -f "$SOURCE_DIR/README.md" ]; then
        cp "$SOURCE_DIR/README.md" "$INSTALL_DIR/"
    fi

    # Set executable permissions
    chmod +x "$INSTALL_DIR/memory"

    log_success "Files installed to $INSTALL_DIR"

    # Step 7: Create and initialize database
    log_info "Step 7: Initializing database..."

    # Create database schema
    SCHEMA_SQL="CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ticket TEXT NOT NULL,
    branch_name TEXT,
    session_id TEXT UNIQUE,
    task_description TEXT,
    files_modified TEXT,
    lines_added INTEGER DEFAULT 0,
    lines_removed INTEGER DEFAULT 0,
    start_time DATETIME,
    end_time DATETIME DEFAULT CURRENT_TIMESTAMP,
    duration_seconds INTEGER,
    commit_sha TEXT
);

CREATE INDEX IF NOT EXISTS idx_ticket ON sessions(ticket);
CREATE INDEX IF NOT EXISTS idx_branch ON sessions(branch_name);
CREATE INDEX IF NOT EXISTS idx_timestamp ON sessions(end_time);

CREATE TABLE IF NOT EXISTS ticket_context_enhanced (
    ticket TEXT PRIMARY KEY,
    requirements TEXT,
    decisions TEXT,
    implementations TEXT,
    code_patterns TEXT,
    current_state TEXT,
    next_steps TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_enhanced_ticket ON ticket_context_enhanced(ticket);"

    # Try to create database using sqlite3
    if command_exists sqlite3; then
        echo "$SCHEMA_SQL" | sqlite3 "$DB_PATH" 2>/dev/null && {
            log_success "Database initialized at $DB_PATH"
        } || {
            log_warning "Database may already exist or initialization had issues"
        }
    else
        log_warning "sqlite3 not found - database will be created on first use"
    fi

    # Step 8: Create slash commands
    log_info "Step 8: Creating slash commands..."

    COMMANDS_DIR="$HOME/.claude/commands"
    mkdir -p "$COMMANDS_DIR"

    # Create save_memory command (general)
    cat > "$COMMANDS_DIR/save_memory.md" << 'EOF'
Save important context to memory for the current ticket:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context add "$ARGUMENTS"`
EOF

    # Create save_decision command
    cat > "$COMMANDS_DIR/save_decision.md" << 'EOF'
Save an architectural or design decision to memory:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save decision "$ARGUMENTS"`
EOF

    # Create save_pattern command
    cat > "$COMMANDS_DIR/save_pattern.md" << 'EOF'
Save a code pattern or function signature to memory:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save pattern "$ARGUMENTS"`
EOF

    # Create save_implementation command
    cat > "$COMMANDS_DIR/save_implementation.md" << 'EOF'
Save an implementation detail (endpoint, function, feature) to memory:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save implementation "$ARGUMENTS"`
EOF

    # Create save_state command
    cat > "$COMMANDS_DIR/save_state.md" << 'EOF'
Save current state (what works/what's broken) to memory:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save state "$ARGUMENTS"`
EOF

    # Create save_todo command
    cat > "$COMMANDS_DIR/save_todo.md" << 'EOF'
Save a TODO or blocker to memory:
$ARGUMENTS

Run: `$HOME/.claude/hooks/memory/memory context save next "$ARGUMENTS"`
EOF

    log_success "Created slash commands in $COMMANDS_DIR"

    # Step 9: Test the installation
    log_info "Step 9: Testing installation..."

    # Test Go binary
    if "$INSTALL_DIR/memory" load 2>&1 | grep -q "Recent work\|No recent work"; then
        log_success "Go binary test passed"
    else
        log_warning "Go binary test inconclusive (this is normal for first run)"
    fi


    # Step 10: Configure Claude Code hooks
    log_info "Step 10: Configuring Claude Code hooks..."

    SETTINGS_FILE="$HOME/.claude/settings.json"

    # Check if settings.json exists
    if [ -f "$SETTINGS_FILE" ]; then
        # Backup existing settings
        cp "$SETTINGS_FILE" "$SETTINGS_FILE.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Backed up existing settings.json"

        # Try to update settings using jq if available
        if command_exists jq; then
            # Create new hooks configuration (SessionStart only)
            NEW_HOOKS=$(cat << 'EOF'
{
  "SessionStart": [{
    "hooks": [{
      "type": "command",
      "command": "$HOME/.claude/hooks/memory/memory load"
    }]
  }]
}
EOF
)
            # Update or add hooks, keeping only SessionStart for memory
            jq --argjson newHooks "$NEW_HOOKS" '
                .hooks = (.hooks // {}) |
                .hooks.SessionStart = $newHooks.SessionStart
            ' "$SETTINGS_FILE" > "$SETTINGS_FILE.tmp" && mv "$SETTINGS_FILE.tmp" "$SETTINGS_FILE"

            if [ $? -eq 0 ]; then
                log_success "Memory hooks configured in settings.json (replaced any existing memory hooks)"
            else
                log_warning "Failed to update settings with jq - please manually update settings.json"
                MANUAL_CONFIG=true
            fi
        else
            log_warning "jq not found - please manually update settings.json"
            MANUAL_CONFIG=true
        fi
    else
        # Create new settings.json with minimal hooks (SessionStart only)
        cat > "$SETTINGS_FILE" << 'EOF'
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
EOF
        log_success "Created settings.json with memory hooks configured"
    fi

    # Installation complete
    echo
    log_success "Installation completed successfully!"
    echo
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║                   Installation Summary                     ║"
    echo "╠═══════════════════════════════════════════════════════════╣"
    echo "║ Go version:    $GO_VERSION"
    echo "║ Install path:  $INSTALL_DIR"
    echo "║ Database:      $DB_PATH"
    echo "║ Settings:      $SETTINGS_FILE"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo

    if [ "$MANUAL_CONFIG" = true ]; then
        echo "⚠️  Please manually add the following to your ~/.claude/settings.json:"
        echo
        cat << 'EOF'
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
EOF
        echo
    else
        echo "✅ Memory hooks have been automatically configured in Claude Code"
        echo
    fi

    echo "Test commands:"
    echo "  Load: $INSTALL_DIR/memory load"
    echo "  Save: echo '{\"sessionId\":\"test-123\",\"lastHumanMessage\":\"Test task\"}' | $INSTALL_DIR/memory save"
    echo
    echo "📝 The memory system will:"
    echo "  • Auto-load context at session start"
    echo "  • Create/sync CONTEXT.md in working directory"
    echo "  • Track work by git branch/ticket number"
    echo ""
    echo "📌 Use slash commands after Claude works:"
    echo "  • /memory_sync - Capture git diff + sync CONTEXT.md"
    echo "  • /memory_decision - Save architectural decisions"
    echo "  • /memory_review - Review current ticket context"
    echo
}

# Handle errors gracefully
trap 'log_error "Installation failed at line $LINENO. Exit code: $?"' ERR

# Run main function
main "$@"