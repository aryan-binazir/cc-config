//go:build sqlite_omit_load_extension

package main

import (
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

func debugLog(format string, args ...interface{}) {
	if os.Getenv("CLAUDE_MEMORY_DEBUG") != "" {
		fmt.Printf("[DEBUG] "+format+"\n", args...)
	}
}

var (
	// Multiple patterns to try in order
	ticketPatterns = []struct {
		name    string
		pattern *regexp.Regexp
	}{
		{"jira", regexp.MustCompile(`(?i)([A-Z]+-\d+)`)}, // JIRA style (case insensitive)
		{"github", regexp.MustCompile(`#(\d+)`)},         // GitHub issue
		{"prefixed", regexp.MustCompile(`(?i)((?:bug|issue|task|story|test)-\d+)`)}, // bug-123, issue-456
		{"feature", regexp.MustCompile(`(?:feature|bugfix|hotfix|fix)/(.+)`)}, // feature branches
		{"release", regexp.MustCompile(`(?:release|v)/?(\d+\.\d+(?:\.\d+)?)`)}, // version branches
	}

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
				fmt.Printf("📦 Synced %d code patterns from git diff for %s\n", len(patterns), ticket)
			} else {
				fmt.Printf("📦 No new code patterns found in git diff\n")
			}
		}
	case "clear":
		if len(os.Args) > 3 {
			clearContext(os.Args[3])
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

func getCurrentBranch() string {
	if output, err := exec.Command("git", "branch", "--show-current").Output(); err == nil {
		return strings.TrimSpace(string(output))
	} else {
		debugLog("Failed to get current git branch: %v", err)
	}
	return ""
}

func extractTicket(branch string) string {
	// Try each pattern in order
	for _, tp := range ticketPatterns {
		if matches := tp.pattern.FindStringSubmatch(branch); len(matches) > 1 {
			return matches[1]
		}
	}

	// If no pattern matches, use the branch name itself as the ticket
	// This ensures EVERY branch gets tracked
	if branch != "" {
		// Clean up the branch name to make a reasonable ticket ID
		ticket := branch
		// Remove common remote prefixes
		ticket = regexp.MustCompile(`^(origin|upstream)/`).ReplaceAllString(ticket, "")
		// Replace special chars with dashes
		ticket = regexp.MustCompile(`[/\\:*?"<>|]+`).ReplaceAllString(ticket, "-")
		// Clean up multiple dashes
		ticket = regexp.MustCompile(`-+`).ReplaceAllString(ticket, "-")
		// Trim dashes from ends
		ticket = strings.Trim(ticket, "-")

		if ticket != "" {
			return ticket
		}
	}

	// Last resort fallback
	return "default"
}

func getModifiedFiles() ([]string, int, int) {
	output, err := exec.Command("git", "diff", "--name-only", "HEAD").Output()
	if err != nil {
		debugLog("Failed to get modified files: %v", err)
		return nil, 0, 0
	}
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(files) == 1 && files[0] == "" {
		files = nil
	}

	output, err = exec.Command("git", "diff", "--numstat", "HEAD").Output()
	if err != nil {
		debugLog("Failed to get git diff numstat: %v", err)
		return files, 0, 0
	}

	var added, removed int
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			if a, err := strconv.Atoi(parts[0]); err == nil {
				added += a
			}
			if r, err := strconv.Atoi(parts[1]); err == nil {
				removed += r
			}
		}
	}
	return files, added, removed
}

func getLastCommitSha() string {
	if output, err := exec.Command("git", "rev-parse", "HEAD").Output(); err == nil {
		return strings.TrimSpace(string(output))
	} else {
		debugLog("Failed to get last commit SHA: %v", err)
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

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

// Context management functions

func loadTicketContext(ticket string) {
	db := openDB()
	defer db.Close()

	var requirements, contextJSON sql.NullString
	err := db.QueryRow(`SELECT requirements, context_points FROM ticket_context WHERE ticket = ?`, ticket).Scan(&requirements, &contextJSON)

	if err == sql.ErrNoRows {
		fmt.Printf("📋 No context saved for %s\n", ticket)
		return
	}
	if err != nil {
		debugLog("Failed to query ticket context for %s: %v", ticket, err)
		os.Exit(0)
	}

	fmt.Printf("\n======================================\n")
	fmt.Printf("📋 %s Context\n", ticket)
	fmt.Printf("======================================\n\n")

	if requirements.Valid && requirements.String != "" {
		fmt.Printf("📜 Requirements:\n%s\n\n", requirements.String)
	}

	if contextJSON.Valid && contextJSON.String != "" {
		var points []ContextPoint
		if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
			debugLog("Failed to parse context JSON in loadTicketContext for ticket %s: %v", ticket, err)
		} else if len(points) > 0 {
			fmt.Printf("📌 Critical Context:\n")
			for _, point := range points {
				if point.IsUserDir {
					fmt.Printf("• 📌 %s\n", point.Text)
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
				fmt.Printf("\n⚠️  Blockers:\n")
				for _, blocker := range blockers {
					fmt.Printf("• %s\n", blocker.Text)
				}
			}
		}
	}

	// Add session summary
	var sessionCount int
	var totalMinutes int
	db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(duration_seconds)/60, 0) FROM sessions WHERE ticket = ?`, ticket).Scan(&sessionCount, &totalMinutes)

	if sessionCount > 0 {
		fmt.Printf("\n📊 Work Summary: %d sessions, %d minutes total\n", sessionCount, totalMinutes)
	}

	fmt.Printf("======================================\n")
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

	if err == nil && contextJSON.Valid && contextJSON.String != "" {
		if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
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
		fmt.Printf("📌 Context saved for %s\n", ticket)
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
	codeBlockRegex := regexp.MustCompile("`([^`]+)`")
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
	funcCallRegex := regexp.MustCompile(`\b(func\s+\w+\([^)]*\)|\w+\([^)]*\))`)
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
			if endpointRegex := regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH)\s+/[^\s]+`); endpointRegex.MatchString(line) {
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
	functionRegex := regexp.MustCompile(`^\+func\s+(\w+)`)
	typeRegex := regexp.MustCompile(`^\+type\s+(\w+)`)
	methodRegex := regexp.MustCompile(`^\+func\s+\([^)]+\)\s+(\w+)`)
	interfaceRegex := regexp.MustCompile(`^\+type\s+(\w+)\s+interface`)

	seenPatterns := make(map[string]bool)

	for _, line := range lines {
		// Skip diff headers
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}

		// Only look at added lines
		if !strings.HasPrefix(line, "+") {
			continue
		}

		// Extract function signatures
		if match := functionRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("func %s", match[1])
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

		// Extract types
		if match := typeRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("type %s", match[1])
			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}

		// Extract interfaces
		if match := interfaceRegex.FindStringSubmatch(line); match != nil {
			pattern := fmt.Sprintf("interface %s", match[1])
			if !seenPatterns[pattern] {
				patterns = append(patterns, pattern)
				seenPatterns[pattern] = true
			}
		}
	}

	return patterns
}



func setRequirements(ticket string, requirements string) {
	db := openDB()
	defer db.Close()

	_, err := db.Exec(`INSERT INTO ticket_context (ticket, requirements, last_updated)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(ticket) DO UPDATE SET
		requirements = excluded.requirements,
		last_updated = CURRENT_TIMESTAMP`, ticket, requirements)

	if err == nil {
		fmt.Printf("📜 Requirements set for %s\n", ticket)
	}
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

	fmt.Printf("📋 Tickets with Context:\n\n")

	for rows.Next() {
		var ticket string
		var requirements, contextJSON sql.NullString
		var lastUpdated time.Time

		if rows.Scan(&ticket, &requirements, &contextJSON, &lastUpdated) != nil {
			continue
		}

		var pointCount int
		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in listTicketsWithContext for ticket %s: %v", ticket, err)
			} else {
				pointCount = len(points)
			}
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
	fmt.Printf("⚠️  Clear all context for %s? This cannot be undone. Type 'yes' to confirm: ", ticket)
	var confirm string
	fmt.Scanln(&confirm)

	if strings.ToLower(confirm) != "yes" {
		fmt.Printf("Cancelled.\n")
		return
	}

	db := openDB()
	defer db.Close()

	_, err := db.Exec(`DELETE FROM ticket_context WHERE ticket = ?`, ticket)
	if err == nil {
		fmt.Printf("✅ Context cleared for %s\n", ticket)
	}
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
		fmt.Printf("✅ No sessions older than %d days to clean up\n", daysToKeep)
		return
	}

	// Delete old sessions
	result, err := db.Exec("DELETE FROM sessions WHERE end_time < ?", cutoffDate)
	if err != nil {
		debugLog("Failed to cleanup old sessions: %v", err)
		fmt.Println("❌ Failed to cleanup old sessions")
		os.Exit(1)
	}

	rowsAffected, _ := result.RowsAffected()

	// Vacuum database to reclaim space
	db.Exec("VACUUM")

	fmt.Printf("✅ Cleaned up %d sessions older than %d days\n", rowsAffected, daysToKeep)

	// Show remaining session count
	var countAfter int
	db.QueryRow("SELECT COUNT(*) FROM sessions").Scan(&countAfter)
	fmt.Printf("📊 Remaining sessions in database: %d\n", countAfter)
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
		if contextJSON.Valid && contextJSON.String != "" {
			json.Unmarshal([]byte(contextJSON.String), &oldPoints)
		}

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
		regexp.MustCompile(`\w+\([^)]*\)`).MatchString(point) {
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
		strings.Contains(point, "✅") || strings.Contains(point, "❌") || strings.Contains(point, "⚠️") {
		return CategoryState
	}

	// Implementations
	if strings.Contains(lowerPoint, "endpoint") || strings.Contains(lowerPoint, "api") ||
		strings.Contains(lowerPoint, "function") || strings.Contains(lowerPoint, "created") ||
		strings.Contains(lowerPoint, "implemented") || strings.Contains(lowerPoint, "added") ||
		regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH)\s+/`).MatchString(point) {
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

func saveEnhancedContext(db *sql.DB, ticket string, requirements string, context *EnhancedContext) error {
	decisionsJSON, _ := json.Marshal(context.Decisions)
	implementationsJSON, _ := json.Marshal(context.Implementations)
	patternsJSON, _ := json.Marshal(context.CodePatterns)
	stateJSON, _ := json.Marshal(context.CurrentState)
	nextJSON, _ := json.Marshal(context.NextSteps)

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
		ticket, requirements,
		string(decisionsJSON), string(implementationsJSON),
		string(patternsJSON), string(stateJSON), string(nextJSON))

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

	if decisionsJSON.Valid && decisionsJSON.String != "" {
		json.Unmarshal([]byte(decisionsJSON.String), &context.Decisions)
	}
	if implementationsJSON.Valid && implementationsJSON.String != "" {
		json.Unmarshal([]byte(implementationsJSON.String), &context.Implementations)
	}
	if patternsJSON.Valid && patternsJSON.String != "" {
		json.Unmarshal([]byte(patternsJSON.String), &context.CodePatterns)
	}
	if stateJSON.Valid && stateJSON.String != "" {
		json.Unmarshal([]byte(stateJSON.String), &context.CurrentState)
	}
	if nextJSON.Valid && nextJSON.String != "" {
		json.Unmarshal([]byte(nextJSON.String), &context.NextSteps)
	}

	return context, requirements.String, nil
}

func loadLegacyContext(db *sql.DB, ticket string) (*EnhancedContext, string, error) {
	var requirements, contextJSON sql.NullString
	err := db.QueryRow(`SELECT requirements, context_points FROM ticket_context WHERE ticket = ?`, ticket).Scan(&requirements, &contextJSON)

	if err != nil {
		return nil, "", err
	}

	var oldPoints []ContextPoint
	if contextJSON.Valid && contextJSON.String != "" {
		json.Unmarshal([]byte(contextJSON.String), &oldPoints)
	}

	enhanced := categorizeOldPoints(oldPoints)
	return enhanced, requirements.String, nil
}

func saveEnhancedContextPoint(ticket string, point string, category ContextCategory, isUserDirective bool) {
	if !isUserDirective && !evaluateEnhancedContextImportance(point, category) {
		return
	}

	db := openDB()
	defer db.Close()

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
	switch category {
	case CategoryDecision:
		if !isDuplicate(context.Decisions, point) {
			context.Decisions = append(context.Decisions, newPoint)
			if len(context.Decisions) > MaxDecisions {
				context.Decisions = consolidateCategoryPoints(context.Decisions, MaxDecisions)
			}
		}
	case CategoryImplementation:
		if !isDuplicate(context.Implementations, point) {
			context.Implementations = append(context.Implementations, newPoint)
			if len(context.Implementations) > MaxImplementations {
				context.Implementations = consolidateCategoryPoints(context.Implementations, MaxImplementations)
			}
		}
	case CategoryPattern:
		if !isDuplicate(context.CodePatterns, point) {
			context.CodePatterns = append(context.CodePatterns, newPoint)
			if len(context.CodePatterns) > MaxCodePatterns {
				context.CodePatterns = consolidateCategoryPoints(context.CodePatterns, MaxCodePatterns)
			}
		}
	case CategoryState:
		if !isDuplicate(context.CurrentState, point) {
			context.CurrentState = append(context.CurrentState, newPoint)
			if len(context.CurrentState) > MaxCurrentState {
				context.CurrentState = consolidateCategoryPoints(context.CurrentState, MaxCurrentState)
			}
		}
	case CategoryNext:
		if !isDuplicate(context.NextSteps, point) {
			context.NextSteps = append(context.NextSteps, newPoint)
			if len(context.NextSteps) > MaxNextSteps {
				context.NextSteps = consolidateCategoryPoints(context.NextSteps, MaxNextSteps)
			}
		}
	}

	// Save back to database
	if err := saveEnhancedContext(db, ticket, requirements, context); err == nil {
		emoji := getCategoryEmoji(category)
		fmt.Printf("%s Context saved for %s (%s)\n", emoji, ticket, category)
	}
}

func getCategoryEmoji(category ContextCategory) string {
	switch category {
	case CategoryDecision:
		return "💡"
	case CategoryImplementation:
		return "🏗️"
	case CategoryPattern:
		return "🔧"
	case CategoryState:
		return "📊"
	case CategoryNext:
		return "📝"
	default:
		return "📌"
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
			strings.Contains(point, "()") || regexp.MustCompile(`\w+\([^)]*\)`).MatchString(point)
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
		return regexp.MustCompile(`(GET|POST|PUT|DELETE|PATCH)\s+/`).MatchString(point) ||
			strings.Contains(lower, "endpoint") || strings.Contains(lower, "api")
	}

	if category == CategoryState {
		// Must be actionable
		return strings.Contains(point, "✅") || strings.Contains(point, "❌") ||
			strings.Contains(point, "⚠️") || strings.Contains(strings.ToLower(point), "working") ||
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
	db := openDB()
	defer db.Close()

	context, requirements, err := loadEnhancedContext(db, ticket)
	if err != nil {
		fmt.Printf("📋 No context saved for %s\n", ticket)
		return
	}

	// Enhanced display format
	fmt.Printf(`
╔════════════════════════════════════════════════════════════
║ 🎯 %s
╚════════════════════════════════════════════════════════════

`, ticket)

	if requirements != "" {
		fmt.Printf("📋 REQUIREMENTS:\n%s\n\n", requirements)
	}

	// Display implementations
	if len(context.Implementations) > 0 {
		fmt.Printf("🏗️ IMPLEMENTATIONS (%d):\n", len(context.Implementations))
		for _, p := range context.Implementations {
			if p.IsUserDir {
				fmt.Printf("  • 📌 %s\n", p.Text)
			} else {
				fmt.Printf("  • %s\n", p.Text)
			}
		}
		fmt.Printf("\n")
	}

	// Display key decisions
	if len(context.Decisions) > 0 {
		fmt.Printf("💡 KEY DECISIONS (%d):\n", len(context.Decisions))
		for _, p := range context.Decisions {
			if p.IsUserDir {
				fmt.Printf("  • 📌 %s\n", p.Text)
			} else {
				fmt.Printf("  • %s\n", p.Text)
			}
		}
		fmt.Printf("\n")
	}

	// Display code patterns
	if len(context.CodePatterns) > 0 {
		fmt.Printf("🔧 CODE PATTERNS (%d):\n", len(context.CodePatterns))
		for _, p := range context.CodePatterns {
			fmt.Printf("  • %s\n", p.Text)
		}
		fmt.Printf("\n")
	}

	// Display current state
	if len(context.CurrentState) > 0 {
		fmt.Printf("📊 CURRENT STATE (%d):\n", len(context.CurrentState))
		for _, p := range context.CurrentState {
			fmt.Printf("  • %s\n", p.Text)
		}
		fmt.Printf("\n")
	}

	// Display next steps
	if len(context.NextSteps) > 0 {
		fmt.Printf("📝 NEXT STEPS (%d):\n", len(context.NextSteps))
		for _, p := range context.NextSteps {
			fmt.Printf("  • %s\n", p.Text)
		}
		fmt.Printf("\n")
	}

	// Add session summary
	var sessionCount int
	var totalMinutes int
	db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(duration_seconds)/60, 0) FROM sessions WHERE ticket = ?`, ticket).Scan(&sessionCount, &totalMinutes)

	if sessionCount > 0 {
		fmt.Printf("📊 Work Summary: %d sessions, %d minutes total\n", sessionCount, totalMinutes)
	}

	fmt.Printf("═══════════════════════════════════════════════════════════\n")
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

	fmt.Printf("📋 Tickets with Enhanced Context:\n\n")

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
			"decisions":       0,
			"implementations": 0,
			"patterns":        0,
			"state":           0,
			"next":            0,
		}

		if decisionsJSON.Valid && decisionsJSON.String != "" {
			var points []ContextPoint
			if json.Unmarshal([]byte(decisionsJSON.String), &points) == nil {
				counts["decisions"] = len(points)
			}
		}
		if implementationsJSON.Valid && implementationsJSON.String != "" {
			var points []ContextPoint
			if json.Unmarshal([]byte(implementationsJSON.String), &points) == nil {
				counts["implementations"] = len(points)
			}
		}
		if patternsJSON.Valid && patternsJSON.String != "" {
			var points []ContextPoint
			if json.Unmarshal([]byte(patternsJSON.String), &points) == nil {
				counts["patterns"] = len(points)
			}
		}
		if stateJSON.Valid && stateJSON.String != "" {
			var points []ContextPoint
			if json.Unmarshal([]byte(stateJSON.String), &points) == nil {
				counts["state"] = len(points)
			}
		}
		if nextJSON.Valid && nextJSON.String != "" {
			var points []ContextPoint
			if json.Unmarshal([]byte(nextJSON.String), &points) == nil {
				counts["next"] = len(points)
			}
		}

		totalPoints := counts["decisions"] + counts["implementations"] + counts["patterns"] +
			counts["state"] + counts["next"]

		fmt.Printf("• %s (%d total: 💡%d 🏗️%d 🔧%d 📊%d 📝%d) - Updated: %s\n",
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

	// Extract function signatures - stop at newline or opening brace
	funcRegex := regexp.MustCompile(`\+?\s*func\s+\w+\s*\([^)]*\)[^\n{]*`)
	if matches := funcRegex.FindAllString(gitDiff, -1); matches != nil {
		for _, match := range matches {
			// Clean up the match
			match = strings.TrimPrefix(match, "+")
			match = strings.TrimSpace(match)
			// Remove everything after the first newline
			if idx := strings.Index(match, "\n"); idx > 0 {
				match = match[:idx]
			}
			// Skip commented lines and ensure it has a function name
			if !strings.Contains(match, "//") && strings.Contains(match, "func ") {
				patterns = append(patterns, strings.TrimSpace(match))
			}
		}
	}

	// Extract type definitions - just the declaration line
	typeRegex := regexp.MustCompile(`\+?\s*type\s+\w+\s+(struct|interface)(\s*{)?`)
	if matches := typeRegex.FindAllString(gitDiff, -1); matches != nil {
		for _, match := range matches {
			match = strings.TrimPrefix(match, "+")
			match = strings.TrimSpace(match)
			if !strings.Contains(match, "//") && strings.Contains(match, "type ") {
				// Just keep the type declaration, not the body
				patterns = append(patterns, strings.TrimSpace(match))
			}
		}
	}

	// Note: Interface method signatures are already captured by funcRegex

	// Extract router patterns
	routeRegex := regexp.MustCompile(`\+?\s*router\.(HandleFunc|Method|Get|Post|Put|Delete)\([^)]+\)`)
	if matches := routeRegex.FindAllString(gitDiff, -1); matches != nil {
		for _, match := range matches {
			match = strings.TrimPrefix(match, "+")
			match = strings.TrimSpace(match)
			patterns = append(patterns, match)
		}
	}

	// Extract API endpoints
	endpointRegex := regexp.MustCompile(`["'/](api|v\d+)?/[^"'\s]+["']`)
	if matches := endpointRegex.FindAllString(gitDiff, -1); matches != nil {
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
	funcNameRegex := regexp.MustCompile(`func\s+([A-Z]\w*)`)
	typeNameRegex := regexp.MustCompile(`type\s+([A-Z]\w*)`)

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
		fmt.Printf("📝 Recent work on %s:\n", ticket)
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
		fmt.Printf("\n📊 Total: %d sessions, %d minutes\n", totalSessions, totalMinutes)
	} else {
		fmt.Printf("📝 No recent work found for %s\n", ticket)
	}

	// Show full enhanced context
	context, requirements, err := loadEnhancedContext(db, ticket)
	if err == nil {
		fmt.Printf("\n╔════════════════════════════════════════════════════════════\n")
		fmt.Printf("║ 🎯 CONTEXT FOR %s\n", ticket)
		fmt.Printf("╚════════════════════════════════════════════════════════════\n\n")

		if requirements != "" {
			fmt.Printf("📋 REQUIREMENTS:\n%s\n\n", requirements)
		}

		// Show all context categories with full details
		if len(context.Decisions) > 0 {
			fmt.Printf("💡 KEY DECISIONS (%d):\n", len(context.Decisions))
			for _, p := range context.Decisions {
				if p.IsUserDir {
					fmt.Printf("  • 📌 %s\n", p.Text)
				} else {
					fmt.Printf("  • %s\n", p.Text)
				}
			}
			fmt.Printf("\n")
		}

		if len(context.Implementations) > 0 {
			fmt.Printf("🏗️ IMPLEMENTATIONS (%d):\n", len(context.Implementations))
			for _, p := range context.Implementations {
				fmt.Printf("  • %s\n", p.Text)
			}
			fmt.Printf("\n")
		}

		if len(context.CodePatterns) > 0 {
			fmt.Printf("🔧 CODE PATTERNS (%d):\n", len(context.CodePatterns))
			for _, p := range context.CodePatterns {
				fmt.Printf("  • %s\n", p.Text)
			}
			fmt.Printf("\n")
		}

		if len(context.CurrentState) > 0 {
			fmt.Printf("📊 CURRENT STATE (%d):\n", len(context.CurrentState))
			for _, p := range context.CurrentState {
				fmt.Printf("  • %s\n", p.Text)
			}
			fmt.Printf("\n")
		}

		if len(context.NextSteps) > 0 {
			fmt.Printf("📝 NEXT STEPS / TODOs (%d):\n", len(context.NextSteps))
			for _, p := range context.NextSteps {
				fmt.Printf("  • %s\n", p.Text)
			}
			fmt.Printf("\n")
		}

		fmt.Printf("═══════════════════════════════════════════════════════════\n")
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

	db := openDB()
	defer db.Close()

	now := time.Now()
	commitSha := getLastCommitSha()

	// Save session data
	var existingID int
	err = db.QueryRow("SELECT id FROM sessions WHERE session_id = ?", data.SessionID).Scan(&existingID)

	if err == sql.ErrNoRows {
		debugLog("Inserting new session: ID=%s, Message=%s", data.SessionID, data.LastHumanMessage)
		_, err = db.Exec(`INSERT INTO sessions (ticket, branch_name, session_id, task_description,
			files_modified, lines_added, lines_removed, start_time, end_time, duration_seconds, commit_sha)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			ticket, branch, data.SessionID, data.LastHumanMessage,
			filesJson, added, removed, now, now, 0, commitSha)
		if err != nil {
			debugLog("Failed to insert session: %v", err)
			fmt.Printf("[Claude Memory Hook] Warning: Failed to save session data: %v\n", err)
		} else {
			debugLog("Session inserted successfully")
		}
	} else if err == nil {
		_, err = db.Exec(`UPDATE sessions SET task_description = ?, files_modified = ?,
			lines_added = ?, lines_removed = ?, end_time = ?, commit_sha = ?
			WHERE session_id = ?`,
			data.LastHumanMessage, filesJson, added, removed, now, commitSha, data.SessionID)
		if err != nil {
			debugLog("Failed to update session: %v", err)
			fmt.Printf("[Claude Memory Hook] Warning: Failed to update session data: %v\n", err)
		}
	}

	// Extract code patterns from git diff
	extractPatternsFromSession()

	// Extract and save context from the message
	if data.LastHumanMessage != "" {
		// Extract user directives (remember:, important:, etc)
		directives := extractUserDirectives(data.LastHumanMessage)
		for _, directive := range directives {
			category := categorizeContext(directive)
			saveEnhancedContextPoint(ticket, directive, category, true)
		}

		// Extract code snippets from message
		codePatterns := extractCodeFromMessage(data.LastHumanMessage)
		for _, pattern := range codePatterns {
			saveEnhancedContextPoint(ticket, pattern, CategoryPattern, false)
		}

		// Extract implementation descriptions
		implementations := extractImplementations(data.LastHumanMessage)
		for _, impl := range implementations {
			saveEnhancedContextPoint(ticket, impl, CategoryImplementation, false)
		}

		// Extract TODOs and blockers
		todos := extractTodos(data.LastHumanMessage)
		for _, todo := range todos {
			debugLog("Extracted TODO: %s", todo)
			saveEnhancedContextPoint(ticket, todo, CategoryNext, false)
		}

		// Extract error states
		if strings.Contains(strings.ToLower(data.LastHumanMessage), "error") ||
		   strings.Contains(strings.ToLower(data.LastHumanMessage), "fails") ||
		   strings.Contains(strings.ToLower(data.LastHumanMessage), "broken") {
			if len(data.LastHumanMessage) < 200 {
				saveEnhancedContextPoint(ticket, data.LastHumanMessage, CategoryState, false)
			}
		}

		// Check if the whole message is important context
		if evaluateContextImportance(data.LastHumanMessage) {
			// Don't duplicate if already extracted
			if len(directives) == 0 && len(codePatterns) == 0 && len(implementations) == 0 && len(todos) == 0 {
				category := categorizeContext(data.LastHumanMessage)
				if len(data.LastHumanMessage) < 200 {
					saveEnhancedContextPoint(ticket, data.LastHumanMessage, category, false)
				}
			}
		}
	}

	fmt.Printf("[Claude Memory Hook] Context saved for ticket: %s\n", ticket)
}
