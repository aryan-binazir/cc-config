//go:build sqlite_omit_load_extension

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func debugLog(format string, args ...interface{}) {
	if os.Getenv("CLAUDE_MEMORY_DEBUG") != "" {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "blockers":
		showBlockers()
	case "todos":
		showTodos()
	case "decisions":
		showDecisions()
	case "directives":
		showUserDirectives()
	case "all":
		showAllTickets()
	case "recent":
		showRecentContext()
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Claude Code Memory Query Tool

Usage: query [command]

Commands:
  blockers    Show all blocked items across tickets
  todos       Show all TODOs and unimplemented features
  decisions   Show technical decisions made
  directives  Show all user directives (📌 marked items)
  all         Show all tickets with context counts
  recent      Show recently updated context (last 7 days)

Examples:
  ./query blockers
  ./query todos
  ./query all
`)
}

func getDBPath() string {
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".claude", "memory.db")
	}
	return ".claude/memory.db"
}

func openDB() *sql.DB {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		debugLog("Failed to open database at %s: %v", dbPath, err)
		os.Exit(0)
	}
	return db
}

func showBlockers() {
	db := openDB()
	defer db.Close()

	// Try enhanced table first
	rows, err := db.Query(`SELECT ticket, next_steps FROM ticket_context_enhanced WHERE next_steps IS NOT NULL`)
	if err != nil {
		// Fall back to legacy table
		rows, err = db.Query(`SELECT ticket, context_points FROM ticket_context WHERE context_points IS NOT NULL`)
		if err != nil {
			debugLog("Failed to query blockers from database: %v", err)
			os.Exit(0)
		}
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "⚠️  Active Blockers and Dependencies:\n\n")

	foundAny := false
	for rows.Next() {
		var ticket string
		var contextJSON sql.NullString

		if rows.Scan(&ticket, &contextJSON) != nil {
			continue
		}

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showBlockers for ticket %s: %v", ticket, err)
			} else {
				for _, point := range points {
					lower := strings.ToLower(point.Text)
					if strings.Contains(lower, "blocked") ||
						strings.Contains(lower, "waiting") ||
						strings.Contains(lower, "depends on") {
						fmt.Fprintf(os.Stderr, "• [%s] %s\n", ticket, point.Text)
						foundAny = true
					}
				}
			}
		}
	}

	if !foundAny {
		fmt.Fprintf(os.Stderr, "No blockers found.\n")
	}
}

func showTodos() {
	db := openDB()
	defer db.Close()

	rows, err := db.Query(`SELECT ticket, context_points FROM ticket_context WHERE context_points IS NOT NULL`)
	if err != nil {
		debugLog("Failed to query todos from database: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "📝 TODOs and Unimplemented Features:\n\n")

	foundAny := false
	for rows.Next() {
		var ticket string
		var contextJSON sql.NullString

		if rows.Scan(&ticket, &contextJSON) != nil {
			continue
		}

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showTodos for ticket %s: %v", ticket, err)
			} else {
				for _, point := range points {
					lower := strings.ToLower(point.Text)
					if strings.Contains(lower, "todo") ||
						strings.Contains(lower, "not implemented") ||
						strings.Contains(lower, "needs implementation") ||
						strings.Contains(lower, "to be done") {
						fmt.Fprintf(os.Stderr, "• [%s] %s\n", ticket, point.Text)
						foundAny = true
					}
				}
			}
		}
	}

	if !foundAny {
		fmt.Fprintf(os.Stderr, "No TODOs found.\n")
	}
}

func showDecisions() {
	db := openDB()
	defer db.Close()

	rows, err := db.Query(`SELECT ticket, context_points FROM ticket_context WHERE context_points IS NOT NULL`)
	if err != nil {
		debugLog("Failed to query decisions from database: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "🎯 Technical Decisions Made:\n\n")

	foundAny := false
	for rows.Next() {
		var ticket string
		var contextJSON sql.NullString

		if rows.Scan(&ticket, &contextJSON) != nil {
			continue
		}

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showDecisions for ticket %s: %v", ticket, err)
			} else {
				for _, point := range points {
					lower := strings.ToLower(point.Text)
					if strings.Contains(lower, "decided to") ||
						strings.Contains(lower, "using") && strings.Contains(lower, "because") ||
						strings.Contains(lower, "chose") ||
						strings.Contains(lower, "instead of") {
						fmt.Fprintf(os.Stderr, "• [%s] %s\n", ticket, point.Text)
						foundAny = true
					}
				}
			}
		}
	}

	if !foundAny {
		fmt.Fprintf(os.Stderr, "No technical decisions found.\n")
	}
}

func showUserDirectives() {
	db := openDB()
	defer db.Close()

	rows, err := db.Query(`SELECT ticket, context_points FROM ticket_context WHERE context_points IS NOT NULL`)
	if err != nil {
		debugLog("Failed to query user directives from database: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "📌 User Directives (Must Follow):\n\n")

	foundAny := false
	for rows.Next() {
		var ticket string
		var contextJSON sql.NullString

		if rows.Scan(&ticket, &contextJSON) != nil {
			continue
		}

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showUserDirectives for ticket %s: %v", ticket, err)
			} else {
				for _, point := range points {
					if point.IsUserDir {
						fmt.Fprintf(os.Stderr, "• [%s] 📌 %s\n", ticket, point.Text)
						foundAny = true
					}
				}
			}
		}
	}

	if !foundAny {
		fmt.Fprintf(os.Stderr, "No user directives found.\n")
	}
}

func showAllTickets() {
	db := openDB()
	defer db.Close()

	rows, err := db.Query(`
		SELECT tc.ticket, tc.requirements, tc.context_points, tc.last_updated,
		       COUNT(s.id) as session_count, COALESCE(SUM(s.duration_seconds)/60, 0) as total_minutes
		FROM ticket_context tc
		LEFT JOIN sessions s ON tc.ticket = s.ticket
		GROUP BY tc.ticket, tc.requirements, tc.context_points, tc.last_updated
		ORDER BY tc.last_updated DESC`)

	if err != nil {
		debugLog("Failed to query all tickets from database: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "📋 All Tickets with Context:\n\n")

	for rows.Next() {
		var ticket string
		var requirements, contextJSON sql.NullString
		var lastUpdated time.Time
		var sessionCount, totalMinutes int

		if rows.Scan(&ticket, &requirements, &contextJSON, &lastUpdated, &sessionCount, &totalMinutes) != nil {
			continue
		}

		var pointCount int
		var userDirCount int
		var blockerCount int

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showAllTickets for ticket %s: %v", ticket, err)
			} else {
				pointCount = len(points)
				for _, point := range points {
					if point.IsUserDir {
						userDirCount++
					}
					lower := strings.ToLower(point.Text)
					if strings.Contains(lower, "blocked") || strings.Contains(lower, "waiting") {
						blockerCount++
					}
				}
			}
		}

		fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(os.Stderr, "🎫 %s\n", ticket)
		fmt.Fprintf(os.Stderr, "  Updated: %s\n", lastUpdated.Format("2006-01-02 15:04"))
		fmt.Fprintf(os.Stderr, "  Sessions: %d (%d minutes total)\n", sessionCount, totalMinutes)
		fmt.Fprintf(os.Stderr, "  Context: %d points", pointCount)

		if userDirCount > 0 {
			fmt.Fprintf(os.Stderr, " (%d user directives)", userDirCount)
		}
		if blockerCount > 0 {
			fmt.Fprintf(os.Stderr, " [%d BLOCKERS]", blockerCount)
		}
		fmt.Fprintf(os.Stderr, "\n")

		if requirements.Valid && requirements.String != "" {
			fmt.Fprintf(os.Stderr, "  Requirements: %s\n", truncate(requirements.String, 70))
		}
	}

	fmt.Fprintf(os.Stderr, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

func showRecentContext() {
	db := openDB()
	defer db.Close()

	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	rows, err := db.Query(`
		SELECT ticket, requirements, context_points, last_updated
		FROM ticket_context
		WHERE last_updated > ?
		ORDER BY last_updated DESC`, sevenDaysAgo)

	if err != nil {
		debugLog("Failed to query recent context from database: %v", err)
		os.Exit(0)
	}
	defer rows.Close()

	fmt.Fprintf(os.Stderr, "📅 Recently Updated Context (Last 7 Days):\n\n")

	foundAny := false
	for rows.Next() {
		var ticket string
		var requirements, contextJSON sql.NullString
		var lastUpdated time.Time

		if rows.Scan(&ticket, &requirements, &contextJSON, &lastUpdated) != nil {
			continue
		}

		fmt.Fprintf(os.Stderr, "\n📋 %s (Updated: %s)\n", ticket, lastUpdated.Format("Jan 2 15:04"))

		if requirements.Valid && requirements.String != "" {
			fmt.Fprintf(os.Stderr, "  Requirements: %s\n", truncate(requirements.String, 70))
		}

		if contextJSON.Valid && contextJSON.String != "" {
			var points []ContextPoint
			if err := json.Unmarshal([]byte(contextJSON.String), &points); err != nil {
				debugLog("Failed to parse context JSON in showRecentContext for ticket %s: %v", ticket, err)
			} else if len(points) > 0 {
				fmt.Fprintf(os.Stderr, "  Recent Context:\n")
				// Show only last 3 points for each ticket
				start := 0
				if len(points) > 3 {
					start = len(points) - 3
				}
				for i := start; i < len(points); i++ {
					point := points[i]
					if point.IsUserDir {
						fmt.Fprintf(os.Stderr, "    • 📌 %s\n", truncate(point.Text, 65))
					} else {
						fmt.Fprintf(os.Stderr, "    • %s\n", truncate(point.Text, 65))
					}
				}
			}
		}
		foundAny = true
	}

	if !foundAny {
		fmt.Fprintf(os.Stderr, "No context updated in the last 7 days.\n")
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
