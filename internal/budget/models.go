// Package budget provides context window budget estimation and visualization.
package budget

import "fmt"

// Model describes an LLM's context window capacity.
type Model struct {
	ID            string // e.g. "claude-sonnet-4-6"
	Name          string // e.g. "Claude Sonnet 4.6"
	Provider      string // e.g. "Anthropic"
	ContextWindow int    // default context window (tokens)
	MaxWindow     int    // max available (e.g. via subscription tier)
}

// WithWindow returns a copy of the model with an overridden context window.
func (m Model) WithWindow(tokens int) Model {
	m.ContextWindow = tokens
	return m
}

// Known models and their context windows.
// DefaultWindow is the baseline; MaxWindow is the max available (e.g. via subscription tier).
var models = map[string]Model{
	// Anthropic
	"claude-opus-4-6":   {ID: "claude-opus-4-6", Name: "Claude Opus 4.6", Provider: "Anthropic", ContextWindow: 200_000, MaxWindow: 1_000_000},
	"claude-sonnet-4-6": {ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", Provider: "Anthropic", ContextWindow: 200_000, MaxWindow: 1_000_000},
	"claude-haiku-4-5":  {ID: "claude-haiku-4-5", Name: "Claude Haiku 4.5", Provider: "Anthropic", ContextWindow: 200_000, MaxWindow: 200_000},

	// OpenAI
	"gpt-4o":      {ID: "gpt-4o", Name: "GPT-4o", Provider: "OpenAI", ContextWindow: 128_000, MaxWindow: 128_000},
	"gpt-4o-mini": {ID: "gpt-4o-mini", Name: "GPT-4o Mini", Provider: "OpenAI", ContextWindow: 128_000, MaxWindow: 128_000},
	"o3":          {ID: "o3", Name: "o3", Provider: "OpenAI", ContextWindow: 200_000, MaxWindow: 200_000},
	"o4-mini":     {ID: "o4-mini", Name: "o4-mini", Provider: "OpenAI", ContextWindow: 200_000, MaxWindow: 200_000},

	// Google
	"gemini-2.5-pro":   {ID: "gemini-2.5-pro", Name: "Gemini 2.5 Pro", Provider: "Google", ContextWindow: 1_000_000, MaxWindow: 1_000_000},
	"gemini-2.5-flash": {ID: "gemini-2.5-flash", Name: "Gemini 2.5 Flash", Provider: "Google", ContextWindow: 1_000_000, MaxWindow: 1_000_000},
}

// LookupModel returns a model by ID.
func LookupModel(id string) (Model, error) {
	m, ok := models[id]
	if !ok {
		return Model{}, fmt.Errorf("unknown model %q (use --list-models to see available models)", id)
	}
	return m, nil
}

// AllModels returns all known models sorted by provider.
func AllModels() []Model {
	out := make([]Model, 0, len(models))
	for _, m := range models {
		out = append(out, m)
	}
	return out
}
