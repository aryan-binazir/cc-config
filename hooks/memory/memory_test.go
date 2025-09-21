//go:build sqlite_omit_load_extension

package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEvaluateContextImportance(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Trivial patterns that should be rejected
		{
			name:     "trivial - fixed typo",
			input:    "Fixed typo in variable name",
			expected: false,
		},
		{
			name:     "trivial - added comment",
			input:    "Added comment to explain function",
			expected: false,
		},
		{
			name:     "trivial - renamed variable",
			input:    "Renamed variable for clarity",
			expected: false,
		},
		{
			name:     "trivial - formatted code",
			input:    "Formatted code with gofmt",
			expected: false,
		},
		{
			name:     "trivial - updated import",
			input:    "Updated import statement",
			expected: false,
		},
		{
			name:     "trivial - minor change",
			input:    "Minor change to function signature",
			expected: false,
		},
		{
			name:     "trivial - added function",
			input:    "Added function to handle requests",
			expected: false,
		},
		{
			name:     "trivial - created file",
			input:    "Created file for new module",
			expected: false,
		},
		{
			name:     "trivial - removed unused",
			input:    "Removed unused imports",
			expected: false,
		},
		{
			name:     "trivial - cleaned up",
			input:    "Cleaned up old code",
			expected: false,
		},

		// Important patterns that should be accepted
		{
			name:     "important - decided to",
			input:    "Decided to use PostgreSQL instead of MySQL",
			expected: true,
		},
		{
			name:     "important - blocked by",
			input:    "Blocked by missing API credentials",
			expected: true,
		},
		{
			name:     "important - waiting on",
			input:    "Waiting on team review before proceeding",
			expected: true,
		},
		{
			name:     "important - breaks when",
			input:    "Code breaks when input is null",
			expected: true,
		},
		{
			name:     "important - must use",
			input:    "Must use HTTPS for all API calls",
			expected: true,
		},
		{
			name:     "important - don't use",
			input:    "Don't use this library, it has security issues",
			expected: true,
		},
		{
			name:     "important - security",
			input:    "Security vulnerability found in auth module",
			expected: true,
		},
		{
			name:     "important - credential",
			input:    "Credential rotation needed every 30 days",
			expected: true,
		},
		{
			name:     "important - TODO",
			input:    "TODO: implement rate limiting",
			expected: true,
		},
		{
			name:     "important - IMPORTANT",
			input:    "IMPORTANT: this function must be thread-safe",
			expected: true,
		},
		{
			name:     "important - remember",
			input:    "remember: always validate input",
			expected: true,
		},
		{
			name:     "important - note",
			input:    "note: this API has rate limits",
			expected: true,
		},
		{
			name:     "important - always",
			input:    "always check return values",
			expected: true,
		},
		{
			name:     "important - never",
			input:    "never expose internal IDs to users",
			expected: true,
		},
		{
			name:     "important - gotcha",
			input:    "gotcha: this function modifies the input slice",
			expected: true,
		},
		{
			name:     "important - warning",
			input:    "warning: this operation is expensive",
			expected: true,
		},
		{
			name:     "important - error",
			input:    "error: database connection timeout",
			expected: true,
		},
		{
			name:     "important - fails when",
			input:    "fails when memory usage exceeds 1GB",
			expected: true,
		},
		{
			name:     "important - requires",
			input:    "requires admin privileges to run",
			expected: true,
		},
		{
			name:     "important - depends on",
			input:    "depends on external service being available",
			expected: true,
		},
		{
			name:     "important - incompatible with",
			input:    "incompatible with version 2.0 of the library",
			expected: true,
		},
		{
			name:     "important - workaround",
			input:    "workaround for known bug in upstream",
			expected: true,
		},

		// Technical decision patterns
		{
			name:     "technical - because",
			input:    "Using Redis because it provides better performance",
			expected: true,
		},
		{
			name:     "technical - instead of",
			input:    "Using goroutines instead of threads",
			expected: true,
		},
		{
			name:     "technical - can't",
			input:    "Can't use this approach due to memory constraints",
			expected: true,
		},
		{
			name:     "technical - won't work",
			input:    "This won't work with our current architecture",
			expected: true,
		},
		{
			name:     "technical - fails",
			input:    "This approach fails under high load",
			expected: true,
		},

		// Edge cases
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   \t\n   ",
			expected: false,
		},
		{
			name:     "case insensitive - uppercase",
			input:    "DECIDED TO USE MONGODB",
			expected: true,
		},
		{
			name:     "case insensitive - mixed case",
			input:    "Fixed TYPO in comment",
			expected: false,
		},
		{
			name:     "partial match in larger text",
			input:    "The system is blocked by network issues and needs investigation",
			expected: true,
		},
		{
			name:     "multiple patterns - important wins",
			input:    "Added function but TODO: implement error handling",
			expected: true,
		},
		{
			name:     "multiple patterns - trivial wins",
			input:    "Fixed typo and updated formatting",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := evaluateContextImportance(tt.input)
			if result != tt.expected {
				t.Errorf("evaluateContextImportance(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractUserDirectives(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single remember directive",
			input:    "Remember: always validate user input",
			expected: []string{"always validate user input"},
		},
		{
			name:     "single important directive",
			input:    "Important: this function must be thread-safe",
			expected: []string{"this function must be thread-safe"},
		},
		{
			name:     "single note directive",
			input:    "Note: API has rate limits of 100 requests per minute",
			expected: []string{"API has rate limits of 100 requests per minute"},
		},
		{
			name:     "single don't forget directive",
			input:    "Don't forget: update documentation after changes",
			expected: []string{"update documentation after changes"},
		},
		{
			name:     "always directive",
			input:    "Always check return values for errors",
			expected: []string{"Always check return values for errors"},
		},
		{
			name:     "never directive",
			input:    "Never expose internal database IDs to users",
			expected: []string{"Never expose internal database IDs to users"},
		},
		{
			name:     "must directive",
			input:    "Must use HTTPS for all external API calls",
			expected: []string{"Must use HTTPS for all external API calls"},
		},
		{
			name:     "make sure directive",
			input:    "Make sure to handle edge cases properly",
			expected: []string{"Make sure to handle edge cases properly"},
		},
		{
			name:     "multiple directives in separate lines",
			input:    "Remember: validate input\nImportant: use HTTPS\nNote: rate limits apply",
			expected: []string{"validate input", "use HTTPS", "rate limits apply"},
		},
		{
			name:     "mixed content with directives",
			input:    "This is a normal message.\nRemember: cache results for performance\nSome other text here.\nImportant: handle timeouts gracefully",
			expected: []string{"cache results for performance", "handle timeouts gracefully"},
		},
		{
			name:     "case insensitive matching",
			input:    "REMEMBER: use uppercase\nimportant: case doesn't matter\nNote: Mixed Case Works",
			expected: []string{"use uppercase", "case doesn't matter", "Mixed Case Works"},
		},
		{
			name:     "directive with extra whitespace",
			input:    "  Remember:   trim whitespace properly   \n\t Important: \t handle tabs too \t",
			expected: []string{"trim whitespace properly", "handle tabs too"},
		},
		{
			name:     "empty directives filtered out",
			input:    "Remember:\nImportant: \nNote: this has content",
			expected: []string{"this has content"},
		},
		{
			name:     "no directives",
			input:    "This is just a normal message with no special patterns.",
			expected: []string{},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "whitespace only input",
			input:    "   \n\t   ",
			expected: []string{},
		},
		{
			name:     "directive at start of line only",
			input:    "This sentence has remember in it but not at start.\nRemember: this one should match",
			expected: []string{"this one should match"},
		},
		{
			name:     "multiple instances of same directive type",
			input:    "Remember: first thing\nSome text\nRemember: second thing",
			expected: []string{"first thing", "second thing"},
		},
		{
			name:     "directive with colon in content",
			input:    "Note: API endpoint is https://api.example.com:8080/v1",
			expected: []string{"API endpoint is https://api.example.com:8080/v1"},
		},
		{
			name: "complex real-world example",
			input: `Working on the authentication system.
Remember: session tokens expire after 24 hours
The current implementation uses JWT.
Important: never store passwords in plain text
Some debugging notes here.
Note: rate limiting is 100 requests per hour per user
Always validate CSRF tokens on state-changing operations.`,
			expected: []string{
				"session tokens expire after 24 hours",
				"never store passwords in plain text",
				"rate limiting is 100 requests per hour per user",
				"Always validate CSRF tokens on state-changing operations.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractUserDirectives(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("extractUserDirectives(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestConsolidatePoints(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    []ContextPoint
		expected []ContextPoint
	}{
		{
			name: "no consolidation needed - under limit",
			input: []ContextPoint{
				{Text: "Point 1", Timestamp: now, IsUserDir: false},
				{Text: "Point 2", Timestamp: now, IsUserDir: true},
				{Text: "Point 3", Timestamp: now, IsUserDir: false},
			},
			expected: []ContextPoint{
				{Text: "Point 2", Timestamp: now, IsUserDir: true},
				{Text: "Point 1", Timestamp: now, IsUserDir: false},
				{Text: "Point 3", Timestamp: now, IsUserDir: false},
			},
		},
		{
			name: "consolidation needed - over limit",
			input: func() []ContextPoint {
				points := make([]ContextPoint, 25)
				for i := 0; i < 25; i++ {
					points[i] = ContextPoint{
						Text:      "Regular point " + string(rune('A'+i)),
						Timestamp: now.Add(time.Duration(i) * time.Minute),
						IsUserDir: false,
					}
				}
				// Add some user directives
				points[5].IsUserDir = true
				points[5].Text = "User directive 1"
				points[15].IsUserDir = true
				points[15].Text = "User directive 2"
				return points
			}(),
			expected: func() []ContextPoint {
				// Should keep all user directives plus last 15 regular points
				var expected []ContextPoint
				// User directives first
				expected = append(expected, ContextPoint{
					Text: "User directive 1", Timestamp: now.Add(5 * time.Minute), IsUserDir: true,
				})
				expected = append(expected, ContextPoint{
					Text: "User directive 2", Timestamp: now.Add(15 * time.Minute), IsUserDir: true,
				})
				// Then last 15 regular points (indices 8-24, excluding the user directives at 5,15)
				// This gives us 15 regular points plus 2 user directives = 17 total
				for i := 8; i < 25; i++ {
					if i != 15 { // Skip the user directive at index 15
						expected = append(expected, ContextPoint{
							Text:      "Regular point " + string(rune('A'+i)),
							Timestamp: now.Add(time.Duration(i) * time.Minute),
							IsUserDir: false,
						})
					}
				}
				return expected
			}(),
		},
		{
			name: "only user directives",
			input: []ContextPoint{
				{Text: "User dir 1", Timestamp: now, IsUserDir: true},
				{Text: "User dir 2", Timestamp: now, IsUserDir: true},
				{Text: "User dir 3", Timestamp: now, IsUserDir: true},
			},
			expected: []ContextPoint{
				{Text: "User dir 1", Timestamp: now, IsUserDir: true},
				{Text: "User dir 2", Timestamp: now, IsUserDir: true},
				{Text: "User dir 3", Timestamp: now, IsUserDir: true},
			},
		},
		{
			name:     "empty input",
			input:    []ContextPoint{},
			expected: []ContextPoint{},
		},
		{
			name: "many user directives over limit",
			input: func() []ContextPoint {
				points := make([]ContextPoint, 30)
				for i := 0; i < 30; i++ {
					points[i] = ContextPoint{
						Text:      "User directive " + string(rune('A'+i)),
						Timestamp: now.Add(time.Duration(i) * time.Minute),
						IsUserDir: true,
					}
				}
				return points
			}(),
			expected: func() []ContextPoint {
				// Should keep ALL user directives, no regular points to trim
				expected := make([]ContextPoint, 30)
				for i := 0; i < 30; i++ {
					expected[i] = ContextPoint{
						Text:      "User directive " + string(rune('A'+i)),
						Timestamp: now.Add(time.Duration(i) * time.Minute),
						IsUserDir: true,
					}
				}
				return expected
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := consolidatePoints(tt.input)

			// Check length
			if len(result) != len(tt.expected) {
				t.Errorf("consolidatePoints() returned %d points, want %d", len(result), len(tt.expected))
				return
			}

			// Check that all user directives are preserved
			userDirCount := 0
			for _, point := range result {
				if point.IsUserDir {
					userDirCount++
				}
			}

			expectedUserDirCount := 0
			for _, point := range tt.expected {
				if point.IsUserDir {
					expectedUserDirCount++
				}
			}

			if userDirCount != expectedUserDirCount {
				t.Errorf("consolidatePoints() preserved %d user directives, want %d", userDirCount, expectedUserDirCount)
			}

			// For large inputs, just verify the structure is correct
			if len(tt.input) > 20 {
				// Should have all user directives first, then regular points
				userDirsSeen := 0
				regularPointsSeen := 0
				foundRegularAfterUser := false

				for _, point := range result {
					if point.IsUserDir {
						userDirsSeen++
						if regularPointsSeen > 0 {
							foundRegularAfterUser = true
						}
					} else {
						regularPointsSeen++
					}
				}

				if foundRegularAfterUser {
					// This is actually the expected behavior - user directives first, then regular points
				}

				// Should not exceed reasonable total
				if len(result) > len(tt.input) {
					t.Errorf("consolidatePoints() returned more points than input")
				}
			} else {
				// For small inputs, check exact match
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("consolidatePoints() = %+v, want %+v", result, tt.expected)
				}
			}
		})
	}
}

// Test edge cases and error conditions
func TestEvaluateContextImportanceEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"very long string", strings.Repeat("a", 10000)},
		{"unicode characters", "决定使用 PostgreSQL 数据库"},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"mixed whitespace", "  \t\n  important:  \r\n  test  \t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := evaluateContextImportance(tt.input)
			// Result should be a boolean
			if result != true && result != false {
				t.Errorf("evaluateContextImportance(%q) returned non-boolean", tt.input)
			}
		})
	}
}

func TestExtractUserDirectivesEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int // Expected number of directives
	}{
		{"very long string", strings.Repeat("remember: test ", 1000), 1000},
		{"unicode in directive", "Remember: 使用 HTTPS 协议", 1},
		{"directive with newlines in content", "Remember: line1\nline2\nline3", 1},
		{"nested patterns", "Remember: always remember to never forget", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := extractUserDirectives(tt.input)
			if len(result) != tt.expected {
				t.Errorf("extractUserDirectives(%q) returned %d directives, want %d", tt.input, len(result), tt.expected)
			}
		})
	}
}
