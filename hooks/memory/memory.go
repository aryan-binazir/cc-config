//go:build sqlite_omit_load_extension

package main

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Add MaxDiffSize to existing constants
const (
	MaxDiffSize = 100 * 1024 // 100KB max for diff storage (~25k tokens)
)

func debugLog(format string, args ...interface{}) {
	if os.Getenv("CLAUDE_MEMORY_DEBUG") != "" {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

var (
	// Removed ticketPatterns - now using branch name directly as ticket ID

	// Fallback pattern for legacy code
	ticketPattern = regexp.MustCompile(`(?i)([A-Z]+-\d+|[a-z]+-\d+)`)

	// Patterns to reject (trivial updates)
	trivialPatterns = []string{
		"fixed typo", "added comment", "renamed variable",
		"formatted code", "updated import", "minor change",
		"added function", "created file", "updated", "modified",
		"changed", "removed unused", "cleaned up",
	}

	// Patterns to accept (important context)
	importantPatterns = []string{
		"decided to", "blocked by", "waiting on", "breaks when",
		"must use", "don't use", "security", "credential",
		"TODO", "IMPORTANT:", "remember:", "note:", "always", "never",
		"gotcha", "warning:", "error:", "fails when", "requires",
		"depends on", "incompatible with", "workaround",
	}

	// User directive patterns
	userDirectivePatterns = []string{
		"remember:", "important:", "don't forget:", "note:",
		"always", "never", "must", "make sure",
	}

	// Compiled regex patterns for extraction
	codeBlockRegex = regexp.MustCompile("`([^`]+)`")
	funcCallRegex = regexp.MustCompile(`\b(func\s+\w+\([^)]*\)|\w+\([^)]*\))`)
	endpointRegex = regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH)\s+/[^\s]+`)
	functionRegex = regexp.MustCompile(`^\+func\s+(\w+)`)
	typeRegex = regexp.MustCompile(`^\+type\s+(\w+)`)
	methodRegex = regexp.MustCompile(`^\+func\s+\([^)]+\)\s+(\w+)`)
	interfaceRegex = regexp.MustCompile(`^\+type\s+(\w+)\s+interface`)
	routeRegex = regexp.MustCompile(`\+?\s*router\.(HandleFunc|Method|Get|Post|Put|Delete)\([^)]+\)`)
	apiEndpointRegex = regexp.MustCompile(`["'/](api|v\d+)?/[^"'\s]+["']`)
	funcExtractRegex = regexp.MustCompile(`\+?\s*func\s+\w+\s*\([^)]*\)[^\n{]*`)
	typeExtractRegex = regexp.MustCompile(`\+?\s*type\s+\w+\s+(struct|interface)(\s*{)?`)
	funcNameRegex = regexp.MustCompile(`func\s+([A-Z]\w*)`)
	typeNameRegex = regexp.MustCompile(`type\s+([A-Z]\w*)`)

	// Other regex patterns used in various places
	itemNumberRegex = regexp.MustCompile(`^(\d+,?)+$|^all$`)
	remotePrefixRegex = regexp.MustCompile(`^(origin|upstream)/`)
	commonBranchesRegex = regexp.MustCompile(`^(main|master|develop|dev|staging|prod|production|release.*)$`)
	codePatternRegex = regexp.MustCompile(`\w+\([^)]*\)`)
	restMethodRegex = regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH)\s+/`)
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(0)
	}

	switch os.Args[1] {
	case "load":
		loadMemory()
	case "save":
		saveMemory()
	case "context":
		handleContext()
	case "cleanup":
		cleanupOldSessions()
	case "extract-ticket":
		// Helper command for slash commands
		if len(os.Args) > 2 {
			ticket := extractTicket(os.Args[2])
			fmt.Print(ticket)
		}
	default:
		os.Exit(0)
	}
}

func handleContext() {
	if len(os.Args) < 3 {
		os.Exit(0)
	}

	switch os.Args[2] {
	case "load":
		if len(os.Args) > 3 {
			loadEnhancedTicketContext(os.Args[3])
		} else {
			branch := getCurrentBranch()
			ticket := extractTicket(branch)
			if ticket != "" {
				loadEnhancedTicketContext(ticket)
			}
		}
	case "add":
		if len(os.Args) > 4 {
			// Auto-categorize if not specified
			category := categorizeContext(os.Args[4])
			saveEnhancedContextPoint(os.Args[3], os.Args[4], category, false)
		}
	case "save":
		// New command format: memory context save <category> <ticket> <point>
		if len(os.Args) > 5 {
			category := ContextCategory(os.Args[3])
			ticket := os.Args[4]
			point := strings.Join(os.Args[5:], " ")
			saveEnhancedContextPoint(ticket, point, category, true)
		} else if len(os.Args) > 4 {
			// Auto-detect ticket from branch
			branch := getCurrentBranch()
			ticket := extractTicket(branch)
			if ticket != "" {
				category := ContextCategory(os.Args[3])
				point := strings.Join(os.Args[4:], " ")
				saveEnhancedContextPoint(ticket, point, category, true)
			}
		}
	case "requirements":
		if len(os.Args) > 4 {
			setRequirements(os.Args[3], strings.Join(os.Args[4:], " "))
		}
	case "list":
		listEnhancedTicketsWithContext()
	case "sync-git":
		// Extract patterns from git diff and save them
		branch := getCurrentBranch()
		ticket := extractTicket(branch)
		if ticket != "" {
			patterns := extractPatternsFromGitDiff()
			for _, pattern := range patterns {
				saveEnhancedContextPoint(ticket, pattern, CategoryPattern, false)
			}
			if len(patterns) > 0 {
				fmt.Printf(" Synced %d code patterns from git diff for %s\n", len(patterns), ticket)
			} else {
				fmt.Printf(" No new code patterns found in git diff\n")
			}
		}
	case "clear":
		if len(os.Args) > 3 {
			clearContext(os.Args[3])
		}
	case "mark-complete":
		// Format: memory context mark-complete <ticket> <number>
		if len(os.Args) > 4 {
			markTodoComplete(os.Args[3], os.Args[4])
		} else {
			fmt.Printf("Error: Please provide TODO number to mark complete\n")
			fmt.Printf("Usage: memory context mark-complete <ticket> <number>\n")
			os.Exit(1)
		}
	case "remove":
		// Format: memory context remove <category> [ticket] [items]
		// or: memory context remove all (nuclear option)
		if len(os.Args) > 3 {
			if os.Args[3] == "all" {
				// Nuclear option
				dropAllTables()
			} else {
				category := os.Args[3]
				var ticket string
				var itemsToRemove string

				// Parse arguments - can be:
				// memory context remove <category> - interactive
				// memory context remove <category> <items> - non-interactive with auto-detected ticket
				// memory context remove <category> <ticket> <items> - non-interactive with explicit ticket
				if len(os.Args) > 4 {
					// Check if arg 4 looks like item numbers (contains digits/commas) or is "all"
					if itemNumberRegex.MatchString(os.Args[4]) {
						// It's items, auto-detect ticket
						branch := getCurrentBranch()
						ticket = extractTicket(branch)
						itemsToRemove = os.Args[4]
					} else {
						// It's a ticket
						ticket = os.Args[4]
						if len(os.Args) > 5 {
							itemsToRemove = os.Args[5]
						}
					}
				} else {
					// Auto-detect from branch
					branch := getCurrentBranch()
					ticket = extractTicket(branch)
				}

				if ticket != "" {
					removeContextItems(ticket, category, itemsToRemove)
				} else {
					fmt.Println("ERROR: No ticket found in branch name")
				}
			}
		}
	}
}

func getDBPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".claude", "memory.db")
	}
	return ".claude/memory.db"
}

func initDB(db *sql.DB) error {
	// Create sessions table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
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
	CREATE INDEX IF NOT EXISTS idx_timestamp ON sessions(end_time);`)

	if err != nil {
		return err
	}

	// Create enhanced ticket_context table with categorized context
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ticket_context_enhanced (
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
	CREATE INDEX IF NOT EXISTS idx_enhanced_ticket ON ticket_context_enhanced(ticket);`)

	if err != nil {
		// Keep legacy table for backward compatibility
		_, err = db.Exec(`CREATE TABLE IF NOT EXISTS ticket_context (
			ticket TEXT PRIMARY KEY,
			requirements TEXT,
			context_points TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_updated DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_context_ticket ON ticket_context(ticket);`)
	}

	// Migrate from old to new schema if needed
	migrateContextData(db)

	return err
}

func openDB() *sql.DB {
	dbPath := getDBPath()
	os.MkdirAll(filepath.Dir(dbPath), 0755)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		debugLog("Failed to open database at %s: %v", dbPath, err)
		os.Exit(0)
	}
	if err := initDB(db); err != nil {
		debugLog("Failed to initialize database: %v", err)
		os.Exit(0)
	}
	return db
}

// withDB executes a function with a database connection, handling open/close automatically
func withDB(fn func(*sql.DB) error) error {
	db := openDB()
	defer db.Close()
	return fn(db)
}

// Helper functions for common patterns

// unmarshalNullableJSON unmarshals a nullable SQL string into a target type
func unmarshalNullableJSON[T any](nullStr sql.NullString, target *T) error {
	if nullStr.Valid && nullStr.String != "" {
		return json.Unmarshal([]byte(nullStr.String), target)
	}
	return nil
}

// execGitCommand executes a git command and returns trimmed output
func execGitCommand(args ...string) (string, error) {
	output, err := exec.Command("git", args...).Output()
	if err != nil {
		debugLog("Git command failed: git %v - %v", args, err)
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// parseNumberList parses comma-separated numbers or "all" into a map of indices
func parseNumberList(input string, max int) map[int]bool {
	result := make(map[int]bool)

	if strings.ToLower(input) == "all" {
		for i := 0; i < max; i++ {
			result[i] = true
		}
		return result
	}

	for _, numStr := range strings.Split(input, ",") {
		numStr = strings.TrimSpace(numStr)
		if num, err := strconv.Atoi(numStr); err == nil && num > 0 && num <= max {
			result[num-1] = true
		}
	}

	return result
}

// countContextPoints counts the number of points in a nullable JSON string
func countContextPoints(jsonStr sql.NullString) int {
	var points []ContextPoint
	if err := unmarshalNullableJSON(jsonStr, &points); err == nil {
		return len(points)
	}
	return 0
}

func getCurrentBranch() string {
	if branch, err := execGitCommand("branch", "--show-current"); err == nil {
		return branch
	}
	return ""
}

func extractTicket(branch string) string {
	if branch == "" {
		return "default"
	}

	// Remove common remote prefixes
	ticket := remotePrefixRegex.ReplaceAllString(branch, "")

	// Skip common branch names that shouldn't be tracked
	// These branches get "default" so no context is saved
	if commonBranchesRegex.MatchString(ticket) {
		return "default"
	}

	if ticket != "" {
		return ticket
	}

	return "default"
}

func getModifiedFiles() ([]string, int, int) {
	// Single git command to get both file names and stats
	output, err := exec.Command("git", "diff", "--numstat", "HEAD").Output()
	if err != nil {
		debugLog("Failed to get git diff numstat: %v", err)
		return nil, 0, 0
	}

	var files []string
	var added, removed int
	seenFiles := make(map[string]bool)

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			// Parse additions and deletions
			if a, err := strconv.Atoi(parts[0]); err == nil {
				added += a
			}
			if r, err := strconv.Atoi(parts[1]); err == nil {
				removed += r
			}
			// Collect unique file names
			fileName := parts[2]
			if !seenFiles[fileName] {
				files = append(files, fileName)
				seenFiles[fileName] = true
			}
		}
	}
	return files, added, removed
}

func getLastCommitSha() string {
	if sha, err := execGitCommand("rev-parse", "HEAD"); err == nil {
		return sha
	}
	return ""
}

func parseTime(timeStr string) time.Time {
	if t, err := time.Parse("2006-01-02 15:04:05", timeStr); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
		return t
	}
	return time.Now()
}

// getSessionStats retrieves session count and total minutes for a ticket
func getSessionStats(db *sql.DB, ticket string) (sessionCount int, totalMinutes int) {
	db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(duration_seconds)/60, 0) FROM sessions WHERE ticket = ?`, ticket).Scan(&sessionCount, &totalMinutes)
	return
}

// displayContext prints formatted context information
func displayContext(context *EnhancedContext, requirements string, ticket string, showBorder bool) {
	if showBorder {
		fmt.Printf(`
============================================================
 CONTEXT FOR %s
============================================================

`, ticket)
	}

	if requirements != "" {
		fmt.Printf(" REQUIREMENTS:\n%s\n\n", requirements)
	}

	// Display implementations
	if len(context.Implementations) > 0 {
		fmt.Printf(" IMPLEMENTATIONS (%d):\n", len(context.Implementations))
		for _, p := range context.Implementations {
			if p.IsUserDir {
				fmt.Printf("  • * %s\n", p.Text)
			} else {
				fmt.Printf("  • %s\n", p.Text)
			}
		}
		fmt.Printf("\n")
	}

	// Display key decisions
	if len(context.Decisions) > 0 {
		fmt.Printf(" KEY DECISIONS (%d):\n", len(context.Decisions))
		for _, p := range context.Decisions {
			if p.IsUserDir {
				fmt.Printf("  • * %s\n", p.Text)
			} else {
				fmt.Printf("  • %s\n", p.Text)
			}
		}
		fmt.Printf("\n")
	}

	// Display code patterns
	if len(context.CodePatterns) > 0 {
		fmt.Printf(" CODE PATTERNS (%d):\n", len(context.CodePatterns))
		for _, p := range context.CodePatterns {
			fmt.Printf("  • %s\n", p.Text)
		}
		fmt.Printf("\n")
	}

	// Display current state
	if len(context.CurrentState) > 0 {
		fmt.Printf(" CURRENT STATE (%d):\n", len(context.CurrentState))
		for _, p := range context.CurrentState {
			fmt.Printf("  • %s\n", p.Text)
		}
		fmt.Printf("\n")
	}

	// Display next steps with numbering
	if len(context.NextSteps) > 0 {
		fmt.Printf(" NEXT STEPS (%d):\n", len(context.NextSteps))
		for i, p := range context.NextSteps {
			fmt.Printf("  %d. %s\n", i+1, p.Text)
		}
		fmt.Printf("\n")
	}

	if showBorder {
		fmt.Printf("============================================================\n")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// Context management functions

func loadTicketContext(ticket string) {
	withDB(func(db *sql.DB) error {
		var requirements, contextJSON sql.NullString
		err := db.QueryRow(`SELECT requirements, context_points FROM ticket_context WHERE ticket = ?`, ticket).Scan(&requirements, &contextJSON)

		if err == sql.ErrNoRows {
			fmt.Printf(" No context saved for %s\n", ticket)
			return nil
		}
		if err != nil {
			debugLog("Failed to query ticket context for %s: %v", ticket, err)
			os.Exit(0)
		}

		fmt.Printf("\n======================================\n")
		fmt.Printf(" %s Context\n", ticket)
		fmt.Printf("======================================\n\n")

		if requirements.Valid && requirements.String != "" {
			fmt.Printf("Requirements:\n%s\n\n", requirements.String)
		}

		var points []ContextPoint
		if err := unmarshalNullableJSON(contextJSON, &points); err != nil {
			debugLog("Failed to parse context JSON in loadTicketContext for ticket %s: %v", ticket, err)
		} else if len(points) > 0 {
				fmt.Printf("* Critical Context:\n")
				for _, point := range points {
					if point.IsUserDir {
						fmt.Printf("• * %s\n", point.Text)
					} else {
						fmt.Printf("• %s\n", point.Text)
					}
				}

				// Look for blockers
				var blockers []ContextPoint
				for _, point := range points {
					lower := strings.ToLower(point.Text)
					if strings.Contains(lower, "blocked") || strings.Contains(lower, "waiting") {
						blockers = append(blockers, point)
					}
				}

				if len(blockers) > 0 {
					fmt.Printf("\nWARNING:  Blockers:\n")
					for _, blocker := range blockers {
						fmt.Printf("• %s\n", blocker.Text)
					}
				}
			}

		// Add session summary
		sessionCount, totalMinutes := getSessionStats(db, ticket)
		if sessionCount > 0 {
			fmt.Printf("\n Work Summary: %d sessions, %d minutes total\n", sessionCount, totalMinutes)
		}

		fmt.Printf("======================================\n")
		return nil
	})
}

func saveContextPoint(ticket string, point string, isUserDirective bool) {
	if !isUserDirective && !evaluateContextImportance(point) {
		return
	}

	db := openDB()
	defer db.Close()

	// Get existing context
	var contextJSON sql.NullString
	var points []ContextPoint

	err := db.QueryRow(`SELECT context_points FROM ticket_context WHERE ticket = ?`, ticket).Scan(&contextJSON)

	if err == nil {
		if err := unmarshalNullableJSON(contextJSON, &points); err != nil {
			debugLog("Failed to parse context JSON for ticket %s: %v", ticket, err)
		}
	}

	// Check for duplicates
	for _, existing := range points {
		if strings.TrimSpace(existing.Text) == strings.TrimSpace(point) {
			return // Already exists
		}
	}

	// Add new point
	newPoint := ContextPoint{
		Text:      point,
		Timestamp: time.Now(),
		IsUserDir: isUserDirective,
	}
	points = append(points, newPoint)

	// Consolidate if too many points (keep backward compatibility)
	if len(points) > MaxTotalPoints {
		points = consolidatePoints(points)
	}

	// Save back
	newJSON, err := json.Marshal(points)
	if err != nil {
		debugLog("Failed to marshal context points for ticket %s: %v", ticket, err)
		return
	}

	_, err = db.Exec(`INSERT INTO ticket_context (ticket, context_points, last_updated)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(ticket) DO UPDATE SET
		context_points = excluded.context_points,
		last_updated = CURRENT_TIMESTAMP`, ticket, string(newJSON))

	if err == nil {
		fmt.Printf("* Context saved for %s\n", ticket)
	}
}

func evaluateContextImportance(point string) bool {
	lower := strings.ToLower(point)

	// Reject trivial patterns
	for _, pattern := range trivialPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return false
		}
	}

	// Accept important patterns
	for _, pattern := range importantPatterns {
		if strings.Contains(lower, strings.ToLower(pattern)) {
			return true
		}
	}

	// Check if it contains technical decisions or gotchas
	if strings.Contains(lower, "because") ||
		strings.Contains(lower, "instead of") ||
		strings.Contains(lower, "can't") ||
		strings.Contains(lower, "won't work") ||
		strings.Contains(lower, "fails") {
		return true
	}

	return false
}

func extractUserDirectives(message string) []string {
	var directives []string
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		lower := strings.ToLower(line)
		for _, pattern := range userDirectivePatterns {
			if strings.Contains(lower, pattern) {
				// Clean up the directive
				directive := strings.TrimSpace(line)
				// Remove common prefixes (case-insensitive)
				lowerDirective := strings.ToLower(directive)
				for _, prefix := range []string{"remember:", "important:", "note:", "don't forget:"} {
					if strings.HasPrefix(lowerDirective, prefix) {
						directive = strings.TrimSpace(directive[len(prefix):])
						break
					}
				}
				directive = strings.TrimSpace(directive)
				if directive != "" && len(directive) < 200 {
					directives = append(directives, directive)
				}
				break
			}
		}
	}

	return directives
}

func extractCodeFromMessage(message string) []string {
	var patterns []string

	// Look for function signatures in backticks
	if matches := codeBlockRegex.FindAllStringSubmatch(message, -1); matches != nil {
		for _, match := range matches {
			code := match[1]
			// Check if it looks like code
			if strings.Contains(code, "(") || strings.Contains(code, "func ") ||
			   strings.Contains(code, "type ") || strings.Contains(code, "interface ") {
				patterns = append(patterns, code)
			}
		}
	}

	// Look for function calls or definitions outside backticks
	if matches := funcCallRegex.FindAllString(message, -1); matches != nil {
		for _, match := range matches {
			if strings.Contains(match, "func ") && len(match) < 100 {
				patterns = append(patterns, match)
			}
		}
	}

	return patterns
}

func extractImplementations(message string) []string {
	var implementations []string
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		lower := strings.ToLower(line)
		// Look for implementation indicators
		if strings.Contains(lower, "implement") || strings.Contains(lower, "create") ||
		   strings.Contains(lower, "add") || strings.Contains(lower, "build") {
			// Look for endpoints
			if endpointRegex.MatchString(line) {
				if match := endpointRegex.FindString(line); match != "" {
					implementations = append(implementations, match)
				}
			} else if len(line) < 150 {
				// Save general implementation description if not too long
				implementations = append(implementations, strings.TrimSpace(line))
			}
		}
	}

	return implementations
}

func extractTodos(message string) []string {
	var todos []string
	lines := strings.Split(message, "\n")

	for _, line := range lines {
		lower := strings.ToLower(line)
		// Look for TODO indicators
		if strings.Contains(lower, "todo") || strings.Contains(lower, "fixme") ||
		   strings.Contains(lower, "blocked") || strings.Contains(lower, "waiting") ||
		   strings.Contains(lower, "need to") || strings.Contains(lower, "should") {
			if len(line) < 150 {
				todos = append(todos, strings.TrimSpace(line))
			}
		}
	}

	return todos
}

// ExtractedContent holds all extracted content from a message
type ExtractedContent struct {
	directives       []string
	codePatterns     []string
	implementations  []string
	todos            []string
	hasErrorState    bool
}

// extractAllFromMessage performs single-pass extraction of all content types
func extractAllFromMessage(message string) *ExtractedContent {
	result := &ExtractedContent{
		directives:      []string{},
		codePatterns:    []string{},
		implementations: []string{},
		todos:           []string{},
		hasErrorState:   false,
	}

	lowerMessage := strings.ToLower(message)
	result.hasErrorState = strings.Contains(lowerMessage, "error") ||
		strings.Contains(lowerMessage, "fails") ||
		strings.Contains(lowerMessage, "broken")

	// Extract code blocks in backticks first (before line processing)
	if matches := codeBlockRegex.FindAllStringSubmatch(message, -1); matches != nil {
		for _, match := range matches {
			code := match[1]
			if strings.Contains(code, "(") || strings.Contains(code, "func ") ||
			   strings.Contains(code, "type ") || strings.Contains(code, "interface ") {
				result.codePatterns = append(result.codePatterns, code)
			}
		}
	}

	// Extract function calls outside backticks
	if matches := funcCallRegex.FindAllString(message, -1); matches != nil {
		for _, match := range matches {
			if strings.Contains(match, "func ") && len(match) < 100 {
				result.codePatterns = append(result.codePatterns, match)
			}
		}
	}

	// Single pass through lines for everything else
	lines := strings.Split(message, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		trimmedLine := strings.TrimSpace(line)

		// Check for user directives
		for _, pattern := range userDirectivePatterns {
			if strings.Contains(lower, pattern) {
				directive := trimmedLine
				// Remove common prefixes
				lowerDirective := strings.ToLower(directive)
				for _, prefix := range []string{"remember:", "important:", "note:", "don't forget:"} {
					if strings.HasPrefix(lowerDirective, prefix) {
						directive = strings.TrimSpace(directive[len(prefix):])
						break
					}
				}
				if directive != "" && len(directive) < 200 && !contains(result.directives, directive) {
					result.directives = append(result.directives, directive)
				}
				break // Move to next line after finding directive
			}
		}

		// Check for implementations
		if strings.Contains(lower, "implement") || strings.Contains(lower, "create") ||
		   strings.Contains(lower, "add") || strings.Contains(lower, "build") {
			if endpointRegex.MatchString(line) {
				if match := endpointRegex.FindString(line); match != "" && !contains(result.implementations, match) {
					result.implementations = append(result.implementations, match)
				}
			} else if len(line) < 150 && !contains(result.implementations, trimmedLine) {
				result.implementations = append(result.implementations, trimmedLine)
			}
		}

		// Check for TODOs
		if strings.Contains(lower, "todo") || strings.Contains(lower, "fixme") ||
		   strings.Contains(lower, "blocked") || strings.Contains(lower, "waiting") ||
		   strings.Contains(lower, "need to") || strings.Contains(lower, "should") {
			if len(line) < 150 && !contains(result.todos, trimmedLine) {
				result.todos = append(result.todos, trimmedLine)
			}
		}
	}

	return result
}

// Helper function to check if slice contains string
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// addContextPointToCategory adds a point to the appropriate category in context
func addContextPointToCategory(context *EnhancedContext, newPoint ContextPoint, category ContextCategory) {
	// Ensure point text isn't too large
	if len(newPoint.Text) > MaxDiffSize {
		truncatedMsg := fmt.Sprintf("[Content truncated: %d bytes -> %d bytes]\n", len(newPoint.Text), MaxDiffSize)
		newPoint.Text = truncatedMsg + newPoint.Text[:MaxDiffSize-len(truncatedMsg)-100] + "\n\n[... truncated ...]"
	}

	switch category {
	case CategoryDecision:
		if !isDuplicate(context.Decisions, newPoint.Text) {
			context.Decisions = append(context.Decisions, newPoint)
			if len(context.Decisions) > MaxDecisions {
				context.Decisions = consolidateCategoryPoints(context.Decisions, MaxDecisions)
			}
		}
	case CategoryImplementation:
		if !isDuplicate(context.Implementations, newPoint.Text) {
			context.Implementations = append(context.Implementations, newPoint)
			if len(context.Implementations) > MaxImplementations {
				context.Implementations = consolidateCategoryPoints(context.Implementations, MaxImplementations)
			}
		}
	case CategoryPattern:
		if !isDuplicate(context.CodePatterns, newPoint.Text) {
			context.CodePatterns = append(context.CodePatterns, newPoint)
			if len(context.CodePatterns) > MaxCodePatterns {
				context.CodePatterns = consolidateCategoryPoints(context.CodePatterns, MaxCodePatterns)
			}
		}
	case CategoryState:
		if !isDuplicate(context.CurrentState, newPoint.Text) {
			context.CurrentState = append(context.CurrentState, newPoint)
			if len(context.CurrentState) > MaxCurrentState {
				context.CurrentState = consolidateCategoryPoints(context.CurrentState, MaxCurrentState)
			}
		}
	case CategoryNext:
		if !isDuplicate(context.NextSteps, newPoint.Text) {
			context.NextSteps = append(context.NextSteps, newPoint)
			if len(context.NextSteps) > MaxNextSteps {
				context.NextSteps = consolidateCategoryPoints(context.NextSteps, MaxNextSteps)
			}
		}
	}
}

// loadEnhancedContextTx loads context within a transaction
func loadEnhancedContextTx(tx *sql.Tx, ticket string) (*EnhancedContext, string, error) {
	var requirements sql.NullString
	var decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON sql.NullString

	err := tx.QueryRow(`SELECT requirements, decisions, implementations, code_patterns, current_state, next_steps
		FROM ticket_context_enhanced WHERE ticket = ?`, ticket).Scan(
		&requirements, &decisionsJSON, &implementationsJSON, &patternsJSON, &stateJSON, &nextJSON)

	if err != nil {
		return nil, "", err
	}

	context := &EnhancedContext{}
	context.UnmarshalFields(decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON)

	return context, requirements.String, nil
}

// saveEnhancedContextTx saves context within a transaction
func saveEnhancedContextTx(tx *sql.Tx, ticket string, requirements string, context *EnhancedContext) error {
	decisions, implementations, patterns, state, next := context.MarshalFields()

	_, err := tx.Exec(`INSERT INTO ticket_context_enhanced
		(ticket, requirements, decisions, implementations, code_patterns, current_state, next_steps, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(ticket) DO UPDATE SET
		requirements = excluded.requirements,
		decisions = excluded.decisions,
		implementations = excluded.implementations,
		code_patterns = excluded.code_patterns,
		current_state = excluded.current_state,
		next_steps = excluded.next_steps,
		last_updated = CURRENT_TIMESTAMP`,
		ticket, requirements, decisions, implementations, patterns, state, next)

	return err
}

func extractPatternsFromGitDiff() []string {
	var patterns []string

	// Get git diff
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.Output()
	if err != nil {
		// Try unstaged diff if no staged changes
		cmd = exec.Command("git", "diff")
		output, err = cmd.Output()
		if err != nil {
			debugLog("Failed to get git diff: %v", err)
			return patterns
		}
	}

	lines := strings.Split(string(output), "\n")
	seenPatterns := make(map[string]bool)

	// Process lines with comment awareness
	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip diff headers
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}

		// Only look at added lines
		if !strings.HasPrefix(line, "+") {
			continue
		}

		cleanLine := strings.TrimSpace(strings.TrimPrefix(line, "+"))

		// Skip pure comment lines
		if strings.HasPrefix(cleanLine, "//") {
			continue
		}

		// Look for preceding comment
		var precedingComment string
		if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "+//") {
			precedingComment = strings.TrimSpace(strings.TrimPrefix(lines[i-1], "+"))
		}

		// Extract function signatures with comments
		if match := functionRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("func %s", match[1])

			// Add inline comment if present
			if idx := strings.Index(cleanLine, "//"); idx > 0 {
				pattern = pattern + " " + strings.TrimSpace(cleanLine[idx:])
			} else if precedingComment != "" {
				pattern = pattern + " " + precedingComment
			}

			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}

		// Extract methods
		if match := methodRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("method %s", match[1])
			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}

		// Extract types with comments
		if match := typeRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("type %s", match[1])

			// Add inline comment if present
			if idx := strings.Index(cleanLine, "//"); idx > 0 {
				pattern = pattern + " " + strings.TrimSpace(cleanLine[idx:])
			} else if precedingComment != "" {
				pattern = pattern + " " + precedingComment
			}

			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}

		// Extract interfaces with comments
		if match := interfaceRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("interface %s", match[1])

			// Add inline comment if present
			if idx := strings.Index(cleanLine, "//"); idx > 0 {
				pattern = pattern + " " + strings.TrimSpace(cleanLine[idx:])
			} else if precedingComment != "" {
				pattern = pattern + " " + precedingComment
			}

			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}
	}

	return patterns
}



func setRequirements(ticket string, requirements string) {
	withDB(func(db *sql.DB) error {
		_, err := db.Exec(`INSERT INTO ticket_context (ticket, requirements, last_updated)
			VALUES (?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(ticket) DO UPDATE SET
			requirements = excluded.requirements,
			last_updated = CURRENT_TIMESTAMP`, ticket, requirements)

		if err == nil {
			fmt.Printf("Requirements set for %s\n", ticket)
		}
		return err
	})
}

func listTicketsWithContext() {
	db := openDB()
	defer db.Close()

	rows, err := db.Query(`SELECT ticket, requirements, context_points, last_updated
		FROM ticket_context ORDER BY last_updated DESC`)
	if err != nil {
		debugLog("Failed to query ticket context list: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Printf(" Tickets with Context:\n\n")

	for rows.Next() {
		var ticket string
		var requirements, contextJSON sql.NullString
		var lastUpdated time.Time

		if rows.Scan(&ticket, &requirements, &contextJSON, &lastUpdated) != nil {
			continue
		}

		var pointCount int
		var points []ContextPoint
		if err := unmarshalNullableJSON(contextJSON, &points); err != nil {
			debugLog("Failed to parse context JSON in listTicketsWithContext for ticket %s: %v", ticket, err)
		} else {
			pointCount = len(points)
		}

		fmt.Printf("• %s (%d context points) - Updated: %s\n",
			ticket, pointCount, lastUpdated.Format("2006-01-02 15:04"))

		if requirements.Valid && requirements.String != "" {
			fmt.Printf("  Requirements: %s\n", truncate(requirements.String, 60))
		}
	}
}

func clearContext(ticket string) {
	// Confirm before clearing
	fmt.Printf("WARNING:  Clear all context for %s? This cannot be undone. Type 'yes' to confirm: ", ticket)

	// Use bufio.Scanner for proper input reading
	scanner := bufio.NewScanner(os.Stdin)
	var confirm string
	if scanner.Scan() {
		confirm = strings.TrimSpace(scanner.Text())
	}

	if strings.ToLower(confirm) != "yes" {
		fmt.Printf("Cancelled.\n")
		return
	}

	withDB(func(db *sql.DB) error {
		_, err := db.Exec(`DELETE FROM ticket_context WHERE ticket = ?`, ticket)
		if err == nil {
			fmt.Printf("SUCCESS: Context cleared for %s\n", ticket)
		}
		return err
	})
}

func markTodoComplete(ticket string, numberStr string) {
	// Parse the TODO number
	todoNum, err := strconv.Atoi(numberStr)
	if err != nil || todoNum < 1 {
		fmt.Printf("Error: Invalid TODO number '%s'\n", numberStr)
		return
	}

	withDB(func(db *sql.DB) error {
		// Load existing context
		context, requirements, err := loadEnhancedContext(db, ticket)
		if err != nil {
			fmt.Printf("No context found for %s\n", ticket)
			return err
		}

		// Check if the number is valid
		if todoNum > len(context.NextSteps) {
			fmt.Printf("Error: TODO #%d not found (only %d TODOs exist)\n", todoNum, len(context.NextSteps))
			return nil
		}

		// Mark the TODO as complete by adding [COMPLETE] prefix if not already present
		idx := todoNum - 1
		if !strings.HasPrefix(context.NextSteps[idx].Text, "[COMPLETE]") {
			context.NextSteps[idx].Text = "[COMPLETE] " + context.NextSteps[idx].Text

			// Save the updated context back
			err = saveEnhancedContext(db, ticket, requirements, context)
			if err != nil {
				fmt.Printf("Error saving updated context: %v\n", err)
				return err
			}

			fmt.Printf(" TODO #%d marked as complete for %s\n", todoNum, ticket)
		} else {
			fmt.Printf(" TODO #%d is already marked as complete\n", todoNum)
		}

		return nil
	})
}

func removeContextItems(ticket string, category string, itemsToRemove string) {
	db := openDB()
	defer db.Close()

	context, _, err := loadEnhancedContext(db, ticket)
	if err != nil {
		fmt.Printf("ERROR: No context found for %s\n", ticket)
		return
	}

	// Map category to the appropriate field
	var items []ContextPoint
	var categoryName string

	switch strings.ToLower(category) {
	case "decisions", "decision":
		items = context.Decisions
		categoryName = "decisions"
	case "implementations", "implementation", "impl":
		items = context.Implementations
		categoryName = "implementations"
	case "patterns", "pattern", "code":
		items = context.CodePatterns
		categoryName = "code_patterns"
	case "state", "status":
		items = context.CurrentState
		categoryName = "current_state"
	case "next", "todo", "todos", "blocker", "blockers":
		items = context.NextSteps
		categoryName = "next_steps"
	default:
		fmt.Printf("ERROR: Invalid category: %s\n", category)
		fmt.Println("Valid categories: decisions, implementations, patterns, state, next")
		return
	}

	if len(items) == 0 {
		fmt.Printf("📭 No %s found for %s\n", categoryName, ticket)
		return
	}

	// Display items with numbers
	fmt.Printf("\n %s for %s:\n", strings.ToUpper(categoryName), ticket)
	for i, item := range items {
		fmt.Printf("%2d. %s\n", i+1, item.Text)
	}

	// Check if items were provided as argument (non-interactive mode)
	var input string
	if itemsToRemove != "" {
		input = itemsToRemove
		fmt.Printf("\nRemoving: %s\n", input)
	} else {
		// Interactive mode
		fmt.Printf("\nEnter numbers to remove (comma-separated), 'all' to remove all, or 'cancel': ")

		// Use bufio.Scanner for proper input reading
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			input = strings.TrimSpace(scanner.Text())
		}

		// If empty input or "cancel", cancel the operation
		if input == "" || strings.ToLower(input) == "cancel" {
			fmt.Println("Cancelled.")
			return
		}
	}

	// Process removal
	var toKeep []ContextPoint

	if strings.ToLower(input) == "all" {
		toKeep = []ContextPoint{}
		fmt.Printf("🗑️  Removing all %s\n", categoryName)
	} else {
		// Parse numbers
		toRemove := parseNumberList(input, len(items))

		// Keep items not marked for removal
		for i, item := range items {
			if !toRemove[i] {
				toKeep = append(toKeep, item)
			}
		}
		fmt.Printf("🗑️  Removing %d items\n", len(toRemove))
	}

	// Update the context
	switch categoryName {
	case "decisions":
		context.Decisions = toKeep
	case "implementations":
		context.Implementations = toKeep
	case "code_patterns":
		context.CodePatterns = toKeep
	case "current_state":
		context.CurrentState = toKeep
	case "next_steps":
		context.NextSteps = toKeep
	}

	// Save the updated context
	err = saveEnhancedContext(db, ticket, "", context)
	if err != nil {
		fmt.Printf("ERROR: Failed to update context: %v\n", err)
		return
	}

	fmt.Printf("SUCCESS: Context updated for %s\n", ticket)
}

func dropAllTables() {
	fmt.Println("WARNING:  WARNING: This will DELETE ALL memory data!")
	fmt.Println("Type 'DELETE EVERYTHING' to confirm: ")

	// Use bufio.Scanner for proper input reading
	scanner := bufio.NewScanner(os.Stdin)
	var confirm string
	if scanner.Scan() {
		confirm = strings.TrimSpace(scanner.Text())
	}

	if confirm != "DELETE EVERYTHING" {
		fmt.Println("Cancelled.")
		return
	}

	db := openDB()
	defer db.Close()

	// Drop all tables
	tables := []string{
		"ticket_context_enhanced",
		"ticket_context",
		"sessions",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
		if err != nil {
			fmt.Printf("ERROR: Failed to drop %s: %v\n", table, err)
			return
		}
	}

	// Recreate tables
	err := initDB(db)
	if err != nil {
		fmt.Printf("ERROR: Failed to recreate tables: %v\n", err)
		return
	}

	fmt.Println("💥 All memory data has been deleted and tables recreated.")
}

func consolidatePoints(points []ContextPoint) []ContextPoint {
	// Keep all user directives
	var consolidated []ContextPoint
	var regular []ContextPoint

	for _, p := range points {
		if p.IsUserDir {
			consolidated = append(consolidated, p)
		} else {
			regular = append(regular, p)
		}
	}

	// Keep most recent 15 regular points
	if len(regular) > 15 {
		regular = regular[len(regular)-15:]
	}

	consolidated = append(consolidated, regular...)
	return consolidated
}

func cleanupOldSessions() {
	// Default to 30 days if not specified
	daysToKeep := 30
	if len(os.Args) > 2 {
		if days, err := strconv.Atoi(os.Args[2]); err == nil && days > 0 {
			daysToKeep = days
		}
	}

	db := openDB()
	defer db.Close()

	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	// Count sessions to be deleted
	var countBefore int
	db.QueryRow("SELECT COUNT(*) FROM sessions WHERE end_time < ?", cutoffDate).Scan(&countBefore)

	if countBefore == 0 {
		fmt.Printf("SUCCESS: No sessions older than %d days to clean up\n", daysToKeep)
		return
	}

	// Delete old sessions
	result, err := db.Exec("DELETE FROM sessions WHERE end_time < ?", cutoffDate)
	if err != nil {
		debugLog("Failed to cleanup old sessions: %v", err)
		fmt.Println("ERROR: Failed to cleanup old sessions")
		os.Exit(1)
	}

	rowsAffected, _ := result.RowsAffected()

	// Vacuum database to reclaim space
	db.Exec("VACUUM")

	fmt.Printf("SUCCESS: Cleaned up %d sessions older than %d days\n", rowsAffected, daysToKeep)

	// Show remaining session count
	var countAfter int
	db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&countAfter)
	fmt.Printf(" Remaining sessions in database: %d\n", countAfter)
}

// Enhanced context functions

func migrateContextData(db *sql.DB) {
	// Check if we have old data to migrate
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM ticket_context").Scan(&count)
	if err != nil || count == 0 {
		return
	}

	rows, err := db.Query("SELECT ticket, requirements, context_points FROM ticket_context")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ticket string
		var requirements, contextJSON sql.NullString
		if rows.Scan(&ticket, &requirements, &contextJSON) != nil {
			continue
		}

		// Parse old context points
		var oldPoints []ContextPoint
		unmarshalNullableJSON(contextJSON, &oldPoints)

		// Categorize old points
		enhanced := categorizeOldPoints(oldPoints)

		// Save to new table
		saveEnhancedContext(db, ticket, requirements.String, enhanced)
	}
}

func categorizeOldPoints(points []ContextPoint) *EnhancedContext {
	enhanced := &EnhancedContext{
		Decisions:       []ContextPoint{},
		Implementations: []ContextPoint{},
		CodePatterns:    []ContextPoint{},
		CurrentState:    []ContextPoint{},
		NextSteps:       []ContextPoint{},
	}

	for _, p := range points {
		p.Category = categorizeContext(p.Text)
		switch p.Category {
		case CategoryDecision:
			enhanced.Decisions = append(enhanced.Decisions, p)
		case CategoryImplementation:
			enhanced.Implementations = append(enhanced.Implementations, p)
		case CategoryPattern:
			enhanced.CodePatterns = append(enhanced.CodePatterns, p)
		case CategoryState:
			enhanced.CurrentState = append(enhanced.CurrentState, p)
		case CategoryNext:
			enhanced.NextSteps = append(enhanced.NextSteps, p)
		}
	}

	return enhanced
}

func categorizeContext(point string) ContextCategory {
	lowerPoint := strings.ToLower(point)

	// Code patterns - check first for code-like content
	if strings.Contains(point, "func ") || strings.Contains(point, "type ") ||
		strings.Contains(point, "struct{") || strings.Contains(point, "interface{") ||
		strings.Contains(point, "()") || strings.Contains(point, "HandleFunc") ||
		codePatternRegex.MatchString(point) {
		return CategoryPattern
	}

	// Next steps
	if strings.Contains(lowerPoint, "todo") || strings.Contains(lowerPoint, "blocked") ||
		strings.Contains(lowerPoint, "waiting") || strings.Contains(lowerPoint, "need to") ||
		strings.Contains(lowerPoint, "next") || strings.Contains(lowerPoint, "pending") {
		return CategoryNext
	}

	// Current state
	if strings.Contains(lowerPoint, "working") || strings.Contains(lowerPoint, "broken") ||
		strings.Contains(lowerPoint, "complete") || strings.Contains(lowerPoint, "fails") ||
		strings.Contains(lowerPoint, "bug") || strings.Contains(lowerPoint, "fixed") ||
		strings.Contains(point, "SUCCESS:") || strings.Contains(point, "ERROR:") || strings.Contains(point, "WARNING:") {
		return CategoryState
	}

	// Implementations
	if strings.Contains(lowerPoint, "endpoint") || strings.Contains(lowerPoint, "api") ||
		strings.Contains(lowerPoint, "function") || strings.Contains(lowerPoint, "created") ||
		strings.Contains(lowerPoint, "implemented") || strings.Contains(lowerPoint, "added") ||
		restMethodRegex.MatchString(point) {
		return CategoryImplementation
	}

	// Decisions
	if strings.Contains(lowerPoint, "decided") || strings.Contains(lowerPoint, "chose") ||
		strings.Contains(lowerPoint, "using") || strings.Contains(lowerPoint, "because") ||
		strings.Contains(lowerPoint, "instead of") || strings.Contains(lowerPoint, "prefer") ||
		strings.Contains(lowerPoint, "will use") {
		return CategoryDecision
	}

	// Default to decision
	return CategoryDecision
}

// MarshalFields converts all context fields to JSON strings
func (ec *EnhancedContext) MarshalFields() (decisions, implementations, patterns, state, next string) {
	decisionsJSON, _ := json.Marshal(ec.Decisions)
	implementationsJSON, _ := json.Marshal(ec.Implementations)
	patternsJSON, _ := json.Marshal(ec.CodePatterns)
	stateJSON, _ := json.Marshal(ec.CurrentState)
	nextJSON, _ := json.Marshal(ec.NextSteps)

	return string(decisionsJSON), string(implementationsJSON),
		string(patternsJSON), string(stateJSON), string(nextJSON)
}

// UnmarshalFields populates context fields from JSON strings
func (ec *EnhancedContext) UnmarshalFields(decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON sql.NullString) {
	unmarshalNullableJSON(decisionsJSON, &ec.Decisions)
	unmarshalNullableJSON(implementationsJSON, &ec.Implementations)
	unmarshalNullableJSON(patternsJSON, &ec.CodePatterns)
	unmarshalNullableJSON(stateJSON, &ec.CurrentState)
	unmarshalNullableJSON(nextJSON, &ec.NextSteps)
}

func saveEnhancedContext(db *sql.DB, ticket string, requirements string, context *EnhancedContext) error {
	decisions, implementations, patterns, state, next := context.MarshalFields()

	_, err := db.Exec(`INSERT INTO ticket_context_enhanced
		(ticket, requirements, decisions, implementations, code_patterns, current_state, next_steps, last_updated)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(ticket) DO UPDATE SET
		requirements = excluded.requirements,
		decisions = excluded.decisions,
		implementations = excluded.implementations,
		code_patterns = excluded.code_patterns,
		current_state = excluded.current_state,
		next_steps = excluded.next_steps,
		last_updated = CURRENT_TIMESTAMP`,
		ticket, requirements, decisions, implementations, patterns, state, next)

	return err
}

func loadEnhancedContext(db *sql.DB, ticket string) (*EnhancedContext, string, error) {
	var requirements sql.NullString
	var decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON sql.NullString

	err := db.QueryRow(`SELECT requirements, decisions, implementations, code_patterns, current_state, next_steps
		FROM ticket_context_enhanced WHERE ticket = ?`, ticket).Scan(
		&requirements, &decisionsJSON, &implementationsJSON, &patternsJSON, &stateJSON, &nextJSON)

	if err != nil {
		// Try legacy table
		return loadLegacyContext(db, ticket)
	}

	context := &EnhancedContext{}
	context.UnmarshalFields(decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON)

	return context, requirements.String, nil
}

func loadLegacyContext(db *sql.DB, ticket string) (*EnhancedContext, string, error) {
	var requirements, contextJSON sql.NullString
	err := db.QueryRow(`SELECT requirements, context_points FROM ticket_context WHERE ticket = ?`, ticket).Scan(&requirements, &contextJSON)

	if err != nil {
		return nil, "", err
	}

	var oldPoints []ContextPoint
	unmarshalNullableJSON(contextJSON, &oldPoints)

	enhanced := categorizeOldPoints(oldPoints)
	return enhanced, requirements.String, nil
}

// addContextPoint adds a new point to the appropriate category list with deduplication and limit enforcement
func addContextPoint(points *[]ContextPoint, newPoint ContextPoint, maxPoints int) bool {
	if isDuplicate(*points, newPoint.Text) {
		return false
	}

	*points = append(*points, newPoint)
	if len(*points) > maxPoints {
		*points = consolidateCategoryPoints(*points, maxPoints)
	}
	return true
}

func saveEnhancedContextPoint(ticket string, point string, category ContextCategory, isUserDirective bool) {
	// Truncate large content to prevent database bloat
	if len(point) > MaxDiffSize {
		truncatedMsg := fmt.Sprintf("[Content truncated: %d bytes -> %d bytes]\n", len(point), MaxDiffSize)
		point = truncatedMsg + point[:MaxDiffSize-len(truncatedMsg)-100] + "\n\n[... truncated ...]"
	}

	if !isUserDirective && !evaluateEnhancedContextImportance(point, category) {
		return
	}

	withDB(func(db *sql.DB) error {
		// Load existing context
		context, requirements, err := loadEnhancedContext(db, ticket)
		if err != nil {
			context = &EnhancedContext{
				Decisions:       []ContextPoint{},
				Implementations: []ContextPoint{},
				CodePatterns:    []ContextPoint{},
				CurrentState:    []ContextPoint{},
				NextSteps:       []ContextPoint{},
			}
		}

		newPoint := ContextPoint{
			Text:      point,
			Category:  category,
			Timestamp: time.Now(),
			IsUserDir: isUserDirective,
		}

		// Add to appropriate category and enforce limits
		var added bool
		switch category {
		case CategoryDecision:
			added = addContextPoint(&context.Decisions, newPoint, MaxDecisions)
		case CategoryImplementation:
			added = addContextPoint(&context.Implementations, newPoint, MaxImplementations)
		case CategoryPattern:
			added = addContextPoint(&context.CodePatterns, newPoint, MaxCodePatterns)
		case CategoryState:
			added = addContextPoint(&context.CurrentState, newPoint, MaxCurrentState)
		case CategoryNext:
			added = addContextPoint(&context.NextSteps, newPoint, MaxNextSteps)
		}

		// Save back to database only if something was added
		if added {
			if err := saveEnhancedContext(db, ticket, requirements, context); err == nil {
				emoji := getCategoryEmoji(category)
				fmt.Printf("%s Context saved for %s (%s)\n", emoji, ticket, category)
			}
			return err
		}
		return nil
	})
}

func getCategoryEmoji(category ContextCategory) string {
	switch category {
	case CategoryDecision:
		return ""
	case CategoryImplementation:
		return ""
	case CategoryPattern:
		return ""
	case CategoryState:
		return ""
	case CategoryNext:
		return ""
	default:
		return "*"
	}
}

func isDuplicate(points []ContextPoint, text string) bool {
	text = strings.TrimSpace(text)
	for _, p := range points {
		if strings.TrimSpace(p.Text) == text {
			return true
		}
	}
	return false
}

func consolidateCategoryPoints(points []ContextPoint, maxPoints int) []ContextPoint {
	// Keep all user directives
	var userDirs []ContextPoint
	var regular []ContextPoint

	for _, p := range points {
		if p.IsUserDir {
			userDirs = append(userDirs, p)
		} else {
			regular = append(regular, p)
		}
	}

	// Keep most recent regular points
	remainingSlots := maxPoints - len(userDirs)
	if remainingSlots < 0 {
		remainingSlots = 0
	}

	if len(regular) > remainingSlots {
		regular = regular[len(regular)-remainingSlots:]
	}

	result := append(userDirs, regular...)
	if len(result) > maxPoints {
		result = result[len(result)-maxPoints:]
	}

	return result
}

func evaluateEnhancedContextImportance(point string, category ContextCategory) bool {
	if category == CategoryPattern {
		// Must contain actual code
		return strings.Contains(point, "func ") || strings.Contains(point, "type ") ||
			strings.Contains(point, "()") || codePatternRegex.MatchString(point)
	}

	if category == CategoryDecision {
		// Must explain reasoning
		lower := strings.ToLower(point)
		return strings.Contains(lower, "because") || strings.Contains(lower, "instead") ||
			strings.Contains(lower, "decided") || strings.Contains(lower, "chose")
	}

	if category == CategoryImplementation {
		// Must be specific
		lower := strings.ToLower(point)
		return restMethodRegex.MatchString(point) ||
			strings.Contains(lower, "endpoint") || strings.Contains(lower, "api")
	}

	if category == CategoryState {
		// Must be actionable
		return strings.Contains(point, "SUCCESS:") || strings.Contains(point, "ERROR:") ||
			strings.Contains(point, "WARNING:") || strings.Contains(strings.ToLower(point), "working") ||
			strings.Contains(strings.ToLower(point), "broken")
	}

	// Always save TODOs and blockers
	if category == CategoryNext {
		return true
	}

	// Use default evaluation for other cases
	return evaluateContextImportance(point)
}

func loadEnhancedTicketContext(ticket string) {
	withDB(func(db *sql.DB) error {
		context, requirements, err := loadEnhancedContext(db, ticket)
		if err != nil {
			fmt.Printf(" No context saved for %s\n", ticket)
			return nil
		}

		displayContext(context, requirements, ticket, true)

		// Add session summary
		sessionCount, totalMinutes := getSessionStats(db, ticket)
		if sessionCount > 0 {
			fmt.Printf(" Work Summary: %d sessions, %d minutes total\n\n", sessionCount, totalMinutes)
			fmt.Printf("============================================================\n")
		}
		return nil
	})
}

func listEnhancedTicketsWithContext() {
	db := openDB()
	defer db.Close()

	// Try enhanced table first
	rows, err := db.Query(`SELECT ticket, requirements, decisions, implementations, code_patterns, current_state, next_steps, last_updated
		FROM ticket_context_enhanced ORDER BY last_updated DESC`)

	if err != nil {
		// Fall back to legacy list
		listTicketsWithContext()
		return
	}
	defer rows.Close()

	fmt.Printf(" Tickets with Enhanced Context:\n\n")

	for rows.Next() {
		var ticket string
		var requirements sql.NullString
		var decisionsJSON, implementationsJSON, patternsJSON, stateJSON, nextJSON sql.NullString
		var lastUpdated time.Time

		if rows.Scan(&ticket, &requirements, &decisionsJSON, &implementationsJSON,
			&patternsJSON, &stateJSON, &nextJSON, &lastUpdated) != nil {
			continue
		}

		// Count points in each category
		counts := map[string]int{
			"decisions":       countContextPoints(decisionsJSON),
			"implementations": countContextPoints(implementationsJSON),
			"patterns":        countContextPoints(patternsJSON),
			"state":           countContextPoints(stateJSON),
			"next":            countContextPoints(nextJSON),
		}

		totalPoints := counts["decisions"] + counts["implementations"] + counts["patterns"] +
			counts["state"] + counts["next"]

		fmt.Printf("• %s (%d total: %d %d %d %d %d) - Updated: %s\n",
			ticket, totalPoints,
			counts["decisions"], counts["implementations"], counts["patterns"],
			counts["state"], counts["next"],
			lastUpdated.Format("2006-01-02 15:04"))

		if requirements.Valid && requirements.String != "" {
			fmt.Printf("  Requirements: %s\n", truncate(requirements.String, 60))
		}
	}
}

// Smart extraction functions for git diffs and code patterns

func extractCodePatternsFromDiff(gitDiff string) []string {
	patterns := []string{}
	lines := strings.Split(gitDiff, "\n")

	// Process lines to extract patterns with their comments
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "+") || strings.Contains(line, "+++") {
			continue
		}

		cleanLine := strings.TrimSpace(strings.TrimPrefix(line, "+"))
		if strings.HasPrefix(cleanLine, "//") {
			continue // Skip pure comment lines
		}

		var comment string
		// Check for preceding comment line
		if i > 0 {
			prevLine := lines[i-1]
			if strings.HasPrefix(strings.TrimSpace(prevLine), "+//") {
				comment = strings.TrimSpace(strings.TrimPrefix(prevLine, "+"))
			}
		}

		// Check for function signatures
		if strings.Contains(cleanLine, "func ") && funcNameRegex.MatchString(cleanLine) {
			pattern := cleanLine
			// Stop at opening brace
			if idx := strings.Index(pattern, "{"); idx > 0 {
				pattern = strings.TrimSpace(pattern[:idx])
			}

			// Extract function name
			if match := funcNameRegex.FindStringSubmatch(pattern); match != nil {
				funcSig := "func " + match[1]

				// Add inline comment if present
				if commentIdx := strings.Index(cleanLine, "//"); commentIdx > 0 {
					funcSig = funcSig + " " + strings.TrimSpace(cleanLine[commentIdx:])
				} else if comment != "" {
					// Add preceding line comment
					funcSig = funcSig + " " + comment
				}

				// Debug output
				if os.Getenv("CLAUDE_MEMORY_DEBUG") != "" {
					debugLog("Function found: %s, Comment: %s, Final: %s", match[1], comment, funcSig)
				}

				patterns = append(patterns, funcSig)
			}
		}

		// Check for type definitions
		if strings.Contains(cleanLine, "type ") && typeNameRegex.MatchString(cleanLine) {
			pattern := cleanLine
			// Stop at opening brace
			if idx := strings.Index(pattern, "{"); idx > 0 {
				pattern = strings.TrimSpace(pattern[:idx])
			}

			// Extract type name
			if match := typeNameRegex.FindStringSubmatch(pattern); match != nil {
				typeSig := "type " + match[1]

				// Add inline comment if present
				if commentIdx := strings.Index(cleanLine, "//"); commentIdx > 0 {
					typeSig = typeSig + " " + strings.TrimSpace(cleanLine[commentIdx:])
				} else if comment != "" {
					// Add preceding line comment
					typeSig = typeSig + " " + comment
				}

				patterns = append(patterns, typeSig)
			}
		}
	}

	// Note: Interface method signatures are already captured by funcExtractRegex

	// Extract router patterns
	if matches := routeRegex.FindAllString(gitDiff, -1); matches != nil {
		for _, match := range matches {
			match = strings.TrimPrefix(match, "+")
			match = strings.TrimSpace(match)
			patterns = append(patterns, match)
		}
	}

	// Extract API endpoints
	if matches := apiEndpointRegex.FindAllString(gitDiff, -1); matches != nil {
		for _, match := range matches {
			patterns = append(patterns, "endpoint: "+match)
		}
	}

	return limitToMostImportant(patterns, 15)
}

func limitToMostImportant(patterns []string, max int) []string {
	// Remove duplicates first
	seen := make(map[string]bool)
	unique := []string{}
	for _, p := range patterns {
		if !seen[p] {
			seen[p] = true
			unique = append(unique, p)
		}
	}

	// Prioritize public functions, types, and APIs
	var publicPatterns []string
	var privatePatterns []string

	for _, p := range unique {
		if startsWithUppercase(p) || strings.Contains(p, "api/") || strings.Contains(p, "endpoint:") {
			publicPatterns = append(publicPatterns, p)
		} else {
			privatePatterns = append(privatePatterns, p)
		}
	}

	result := publicPatterns
	remaining := max - len(result)
	if remaining > 0 && len(privatePatterns) > 0 {
		if len(privatePatterns) <= remaining {
			result = append(result, privatePatterns...)
		} else {
			result = append(result, privatePatterns[:remaining]...)
		}
	}

	if len(result) > max {
		result = result[:max]
	}

	return result
}

func startsWithUppercase(s string) bool {
	// Check if pattern starts with uppercase (public in Go)
	return funcNameRegex.MatchString(s) || typeNameRegex.MatchString(s)
}

func extractPatternsFromSession() {
	// Get git diff
	output, err := exec.Command("git", "diff", "HEAD").Output()
	if err != nil {
		return
	}

	gitDiff := string(output)
	if gitDiff == "" {
		// Try staged changes
		output, err = exec.Command("git", "diff", "--cached").Output()
		if err != nil {
			return
		}
		gitDiff = string(output)
	}

	if gitDiff == "" {
		return
	}

	patterns := extractCodePatternsFromDiff(gitDiff)

	// Auto-save important patterns
	branch := getCurrentBranch()
	ticket := extractTicket(branch)
	if ticket == "" {
		return
	}

	for _, pattern := range patterns {
		if strings.Contains(pattern, "func ") || strings.Contains(pattern, "type ") ||
			strings.Contains(pattern, "endpoint:") {
			saveEnhancedContextPoint(ticket, pattern, CategoryPattern, false)
		}
	}
}

// Original functions with context integration

func loadMemory() {
	branch := getCurrentBranch()
	ticket := extractTicket(branch)
	if ticket == "" {
		fmt.Println("[Claude Memory Hook] No ticket found in branch name")
		os.Exit(0)
	}

	db := openDB()
	defer db.Close()

	// First show recent sessions
	rows, err := db.Query(`SELECT task_description, start_time, duration_seconds, files_modified
		FROM sessions WHERE ticket = ? ORDER BY end_time DESC LIMIT 5`, ticket)
	if err != nil {
		debugLog("Failed to query recent sessions for ticket %s: %v", ticket, err)
		os.Exit(0)
	}
	defer rows.Close()

	var sessions []struct {
		desc     string
		start    time.Time
		duration int
		files    []string
	}
	totalSessions, totalMinutes := 0, 0

	for rows.Next() {
		var desc, timeStr, filesJson string
		var duration int
		if rows.Scan(&desc, &timeStr, &duration, &filesJson) != nil {
			continue
		}

		var files []string
		if filesJson != "" {
			json.Unmarshal([]byte(filesJson), &files)
		}

		sessions = append(sessions, struct {
			desc     string
			start    time.Time
			duration int
			files    []string
		}{desc, parseTime(timeStr), duration, files})
		totalSessions++
		totalMinutes += duration / 60
	}

	if len(sessions) > 0 {
		fmt.Printf(" Recent work on %s:\n", ticket)
		for _, s := range sessions {
			duration := s.duration / 60
			if duration == 0 {
				duration = 1
			}
			// Build description from task description and files
			desc := s.desc
			if desc == "" && len(s.files) > 0 {
				// If no description but files modified, show files
				desc = fmt.Sprintf("Modified: %s", strings.Join(s.files, ", "))
			} else if desc == "" {
				desc = "[Session started - no description yet]"
			}
			fmt.Printf("  • %s (%dm): %s\n",
				s.start.Format("2006-01-02 15:04"), duration, truncate(desc, 60))
		}
		fmt.Printf("\n Total: %d sessions, %d minutes\n", totalSessions, totalMinutes)
	} else {
		fmt.Printf(" No recent work found for %s\n", ticket)
	}

	// Show full enhanced context
	context, requirements, err := loadEnhancedContext(db, ticket)
	if err == nil {
		// Use custom header for "CONTEXT FOR" instead of just ticket name
		fmt.Printf("\n============================================================\n")
		fmt.Printf(" CONTEXT FOR %s\n", ticket)
		fmt.Printf("============================================================\n\n")

		// Display using helper (without border since we have custom header)
		displayContext(context, requirements, ticket, false)
		fmt.Printf("============================================================\n")
	}
}

func saveMemory() {
	fmt.Println("[Claude Memory Hook] Saving session context...")

	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		debugLog("Failed to read input from stdin: %v", err)
		fmt.Println("[Claude Memory Hook] Failed to save context")
		os.Exit(0)
	}

	var data InputData
	if err := json.Unmarshal(input, &data); err != nil {
		debugLog("Failed to parse JSON input: %v", err)
		fmt.Println("[Claude Memory Hook] Failed to save context")
		os.Exit(0)
	}

	debugLog("Parsed JSON - SessionID: '%s', LastHumanMessage: '%s'", data.SessionID, data.LastHumanMessage)

	branch := getCurrentBranch()
	ticket := extractTicket(branch)
	if ticket == "" {
		os.Exit(0)
	}

	// Generate session ID if empty
	if data.SessionID == "" {
		data.SessionID = fmt.Sprintf("session-%d-%d", time.Now().Unix(), os.Getpid())
		debugLog("Generated session ID: %s", data.SessionID)
	}

	files, added, removed := getModifiedFiles()
	filesJson := ""
	if len(files) > 0 {
		if data, err := json.Marshal(files); err == nil {
			filesJson = string(data)
		}
	}

	// Use database transaction for all operations
	withDB(func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		now := time.Now()
		commitSha := getLastCommitSha()

		// Save session data in transaction
		var existingID int
		err = tx.QueryRow("SELECT id FROM sessions WHERE session_id = ?", data.SessionID).Scan(&existingID)

		if err == sql.ErrNoRows {
			debugLog("Inserting new session: ID=%s, Message=%s", data.SessionID, data.LastHumanMessage)
			_, err = tx.Exec(`INSERT INTO sessions (ticket, branch_name, session_id, task_description,
				files_modified, lines_added, lines_removed, start_time, end_time, duration_seconds, commit_sha)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				ticket, branch, data.SessionID, data.LastHumanMessage,
				filesJson, added, removed, now, now, 0, commitSha)
			if err != nil {
				debugLog("Failed to insert session: %v", err)
				return err
			}
			debugLog("Session inserted successfully")
		} else if err == nil {
			_, err = tx.Exec(`UPDATE sessions SET task_description = ?, files_modified = ?,
				lines_added = ?, lines_removed = ?, end_time = ?, commit_sha = ?
				WHERE session_id = ?`,
				data.LastHumanMessage, filesJson, added, removed, now, commitSha, data.SessionID)
			if err != nil {
				debugLog("Failed to update session: %v", err)
				return err
			}
		}

		// Extract and save context from the message using single-pass extraction
		if data.LastHumanMessage != "" {
			extractedContext := extractAllFromMessage(data.LastHumanMessage)

			// Load existing context within transaction
			context, requirements, err := loadEnhancedContextTx(tx, ticket)
			if err != nil {
				context = &EnhancedContext{
					Decisions:       []ContextPoint{},
					Implementations: []ContextPoint{},
					CodePatterns:    []ContextPoint{},
					CurrentState:    []ContextPoint{},
					NextSteps:       []ContextPoint{},
				}
				requirements = ""
			}

			// Add all extracted context points
			for _, directive := range extractedContext.directives {
				category := categorizeContext(directive)
				newPoint := ContextPoint{
					Text:      directive,
					Category:  category,
					Timestamp: time.Now(),
					IsUserDir: true,
				}
				addContextPointToCategory(context, newPoint, category)
			}

			for _, pattern := range extractedContext.codePatterns {
				newPoint := ContextPoint{
					Text:      pattern,
					Category:  CategoryPattern,
					Timestamp: time.Now(),
					IsUserDir: false,
				}
				addContextPointToCategory(context, newPoint, CategoryPattern)
			}

			for _, impl := range extractedContext.implementations {
				newPoint := ContextPoint{
					Text:      impl,
					Category:  CategoryImplementation,
					Timestamp: time.Now(),
					IsUserDir: false,
				}
				addContextPointToCategory(context, newPoint, CategoryImplementation)
			}

			for _, todo := range extractedContext.todos {
				debugLog("Extracted TODO: %s", todo)
				newPoint := ContextPoint{
					Text:      todo,
					Category:  CategoryNext,
					Timestamp: time.Now(),
					IsUserDir: false,
				}
				addContextPointToCategory(context, newPoint, CategoryNext)
			}

			// Check for error states
			if extractedContext.hasErrorState && len(data.LastHumanMessage) < 200 {
				message := data.LastHumanMessage
				if len(message) > MaxDiffSize {
					truncatedMsg := fmt.Sprintf("[Content truncated: %d bytes -> %d bytes]\n", len(message), MaxDiffSize)
					message = truncatedMsg + message[:MaxDiffSize-len(truncatedMsg)-100] + "\n\n[... truncated ...]"
				}
				newPoint := ContextPoint{
					Text:      message,
					Category:  CategoryState,
					Timestamp: time.Now(),
					IsUserDir: false,
				}
				addContextPointToCategory(context, newPoint, CategoryState)
			}

			// Save the updated context in the transaction
			if err := saveEnhancedContextTx(tx, ticket, requirements, context); err != nil {
				return err
			}
		}

		// Commit transaction
		return tx.Commit()
	})

	// Extract code patterns from git diff (outside transaction)
	extractPatternsFromSession()

	fmt.Printf("[Claude Memory Hook] Context saved for ticket: %s\n", ticket)
}
