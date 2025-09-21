//go:build sqlite_omit_load_extension

package main

import (
	"strings"
	"testing"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "string shorter than limit",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "string equal to limit",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "string longer than limit",
			input:    "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   5,
			expected: "",
		},
		{
			name:     "single character string",
			input:    "a",
			maxLen:   1,
			expected: "a",
		},
		{
			name:     "truncate to very small limit",
			input:    "hello world",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "truncate to exactly ellipsis length",
			input:    "hello",
			maxLen:   3,
			expected: "...",
		},
		{
			name:     "long technical message",
			input:    "Failed to connect to database server at hostname:5432 with error: connection timeout",
			maxLen:   50,
			expected: "Failed to connect to database server at hostna...",
		},
		{
			name:     "unicode characters",
			input:    "数据库连接失败，请检查网络设置",
			maxLen:   10,
			expected: "数据库连接失败，请...",
		},
		{
			name:     "whitespace handling",
			input:    "  hello world  ",
			maxLen:   10,
			expected: "  hello...",
		},
		{
			name:     "newline in string",
			input:    "line1\nline2\nline3",
			maxLen:   8,
			expected: "line1...",
		},
		{
			name:     "special characters",
			input:    "error: 500 - internal server error!",
			maxLen:   15,
			expected: "error: 500 -...",
		},
		{
			name:     "zero length limit",
			input:    "hello",
			maxLen:   0,
			expected: "...",
		},
		{
			name:     "negative length limit",
			input:    "hello",
			maxLen:   -5,
			expected: "...",
		},
		{
			name:     "very long string with small limit",
			input:    strings.Repeat("a", 1000),
			maxLen:   10,
			expected: "aaaaaaa...",
		},
		{
			name:     "very long string with reasonable limit",
			input:    strings.Repeat("a", 1000),
			maxLen:   100,
			expected: strings.Repeat("a", 97) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}

			// Additional validation: result should never exceed maxLen
			if len(result) > tt.maxLen && tt.maxLen > 0 {
				t.Errorf("truncate(%q, %d) returned string of length %d, which exceeds maxLen %d",
					tt.input, tt.maxLen, len(result), tt.maxLen)
			}

			// If input was shorter than limit, result should be unchanged
			if len(tt.input) <= tt.maxLen && result != tt.input {
				t.Errorf("truncate(%q, %d) = %q, should be unchanged when input is shorter than limit",
					tt.input, tt.maxLen, result)
			}
		})
	}
}

// Test edge cases for truncate function
func TestTruncateEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		maxLen      int
		description string
	}{
		{
			name:        "maxLen 1",
			input:       "hello",
			maxLen:      1,
			description: "Should handle very small maxLen gracefully",
		},
		{
			name:        "maxLen 2",
			input:       "hello",
			maxLen:      2,
			description: "Should handle maxLen smaller than ellipsis",
		},
		{
			name:        "empty with zero limit",
			input:       "",
			maxLen:      0,
			description: "Empty string with zero limit",
		},
		{
			name:        "tab and newline characters",
			input:       "hello\tworld\ntest",
			maxLen:      8,
			description: "Should handle whitespace characters",
		},
		{
			name:        "only spaces",
			input:       "     ",
			maxLen:      3,
			description: "String containing only spaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := truncate(tt.input, tt.maxLen)

			// Basic sanity checks
			if tt.maxLen > 0 && len(result) > tt.maxLen {
				t.Errorf("truncate(%q, %d) returned string longer than limit: %q",
					tt.input, tt.maxLen, result)
			}

			// If input is empty, result should be empty (unless there's a bug)
			if tt.input == "" && result != "" && tt.maxLen > 0 {
				t.Errorf("truncate(%q, %d) should return empty string for empty input, got %q",
					tt.input, tt.maxLen, result)
			}
		})
	}
}

// Test realistic scenarios with actual error messages and context
func TestTruncateRealisticScenarios(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "database error message",
			input:    "Failed to connect to PostgreSQL database at localhost:5432 - connection refused",
			maxLen:   60,
			expected: "Failed to connect to PostgreSQL database at localho...",
		},
		{
			name:     "API response truncation",
			input:    "HTTP 500 Internal Server Error: The server encountered an unexpected condition",
			maxLen:   40,
			expected: "HTTP 500 Internal Server Error: The s...",
		},
		{
			name:     "file path truncation",
			input:    "/usr/local/bin/very/long/path/to/some/file/that/needs/truncation.txt",
			maxLen:   30,
			expected: "/usr/local/bin/very/long/pa...",
		},
		{
			name:     "JSON error message",
			input:    `{"error": "validation failed", "details": "missing required field 'username'"}`,
			maxLen:   45,
			expected: `{"error": "validation failed", "details":...`,
		},
		{
			name:     "git commit message",
			input:    "Add comprehensive unit tests for memory system functions including edge cases",
			maxLen:   50,
			expected: "Add comprehensive unit tests for memory system...",
		},
		{
			name:     "user directive",
			input:    "Remember: always validate user input before processing to prevent security vulnerabilities",
			maxLen:   70,
			expected: "Remember: always validate user input before processing to pr...",
		},
		{
			name:     "technical decision",
			input:    "Decided to use Redis for caching because it provides better performance than in-memory solutions",
			maxLen:   80,
			expected: "Decided to use Redis for caching because it provides better performance...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// Benchmark test to ensure truncate function performs well
func BenchmarkTruncate(b *testing.B) {
	testCases := []struct {
		name   string
		input  string
		maxLen int
	}{
		{"short_string", "hello world", 20},
		{"medium_string", strings.Repeat("test ", 50), 100},
		{"long_string", strings.Repeat("benchmark test string ", 1000), 200},
		{"very_long_string", strings.Repeat("x", 10000), 50},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				truncate(tc.input, tc.maxLen)
			}
		})
	}
}

// Test that truncate maintains string validity (no broken UTF-8)
func TestTruncateUTF8Safety(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
	}{
		{
			name:   "chinese characters",
			input:  "这是一个测试字符串，包含中文字符",
			maxLen: 15,
		},
		{
			name:   "emoji",
			input:  "Hello 👋 World 🌍 Test 🧪",
			maxLen: 10,
		},
		{
			name:   "mixed unicode",
			input:  "Café naïve résumé 测试 🚀",
			maxLen: 12,
		},
		{
			name:   "cyrillic",
			input:  "Привет мир это тест",
			maxLen: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)

			// Check that result is valid UTF-8
			if strings.ToValidUTF8(result, "") != result {
				t.Errorf("truncate(%q, %d) produced invalid UTF-8: %q", tt.input, tt.maxLen, result)
			}

			// Length check
			if len(result) > tt.maxLen {
				t.Errorf("truncate(%q, %d) exceeded maxLen: got %d chars", tt.input, tt.maxLen, len(result))
			}
		})
	}
}
