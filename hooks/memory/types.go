//go:build sqlite_omit_load_extension

package main

import "time"

// Constants for memory management
const (
	MaxTotalPoints     = 50
	MaxDecisions       = 10
	MaxImplementations = 15
	MaxCodePatterns    = 15
	MaxCurrentState    = 10
	MaxNextSteps       = 10
)

// ContextCategory defines the type of context
type ContextCategory string

const (
	CategoryDecision       ContextCategory = "decision"
	CategoryImplementation ContextCategory = "implementation"
	CategoryPattern        ContextCategory = "pattern"
	CategoryState          ContextCategory = "state"
	CategoryNext           ContextCategory = "next"
)

// ContextPoint represents a piece of context information for a ticket
type ContextPoint struct {
	Text      string          `json:"text"`
	Category  ContextCategory `json:"category"`
	Timestamp time.Time       `json:"timestamp"`
	IsUserDir bool            `json:"is_user_directive"`
}

// EnhancedContext represents categorized context for a ticket
type EnhancedContext struct {
	Decisions       []ContextPoint `json:"decisions"`
	Implementations []ContextPoint `json:"implementations"`
	CodePatterns    []ContextPoint `json:"code_patterns"`
	CurrentState    []ContextPoint `json:"current_state"`
	NextSteps       []ContextPoint `json:"next_steps"`
}

// EnhancedContextItem represents an item in the enhanced context JSON
type EnhancedContextItem struct {
	Text             string    `json:"text"`
	Timestamp        time.Time `json:"timestamp"`
	IsUserDirective  bool      `json:"is_user_directive,omitempty"`
}

// InputData represents the data passed to the save command
type InputData struct {
	SessionID        string `json:"sessionId,omitempty"`
	LastHumanMessage string `json:"lastHumanMessage,omitempty"`
}
