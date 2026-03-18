// Package transcript parses Claude Code session transcripts (.jsonl)
// and reconstructs context window usage from real API token counts.
package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Session represents a parsed transcript with context window analysis.
type Session struct {
	SessionID    string     `json:"session_id"`
	Model        string     `json:"model"`
	Turns        []Turn     `json:"turns"`
	FilesRead    []FileRead `json:"files_read"`
	TotalInput   int        `json:"total_input_tokens"`
	TotalOutput  int        `json:"total_output_tokens"`
	TotalCached  int        `json:"total_cached_tokens"`
	PeakContext  int        `json:"peak_context_tokens"`
	WindowSize   int        `json:"window_size"` // inferred from model
}

// Turn represents one assistant response with its token usage.
type Turn struct {
	Index        int    `json:"index"`
	InputTokens  int    `json:"input_tokens"`
	CacheCreate  int    `json:"cache_creation_tokens"`
	CacheRead    int    `json:"cache_read_tokens"`
	OutputTokens int    `json:"output_tokens"`
	ContextSize  int    `json:"context_size"`  // total tokens in window at this turn
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	TextPreview  string `json:"text_preview,omitempty"`
}

// ToolCall represents a tool invocation in a turn.
type ToolCall struct {
	Name  string `json:"name"`
	Input string `json:"input"` // abbreviated
}

// FileRead tracks a file that was loaded into context.
type FileRead struct {
	Path       string `json:"path"`
	TurnIndex  int    `json:"turn_index"`
	TokensEst  int    `json:"tokens_est"` // estimated from content length
}

// raw JSON structures for parsing transcript lines.
type rawLine struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionId"`
	Message   json.RawMessage `json:"message,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	ToolUseResult interface{} `json:"toolUseResult,omitempty"`
}

type rawMessage struct {
	Type    string           `json:"type"`
	Model   string           `json:"model"`
	Content []rawContentBlock `json:"content"`
	Usage   rawUsage         `json:"usage"`
}

type rawContentBlock struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	// tool_result fields
	Content json.RawMessage `json:"content,omitempty"`
}

type rawUsage struct {
	InputTokens             int `json:"input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens    int `json:"cache_read_input_tokens"`
	OutputTokens            int `json:"output_tokens"`
}

// Parse reads a transcript JSONL file and returns a Session analysis.
func Parse(path string) (*Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}
	defer f.Close()

	s := &Session{}
	turnIndex := 0

	scanner := bufio.NewScanner(f)
	// Increase buffer for large lines.
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		var line rawLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue // skip unparseable lines
		}

		if s.SessionID == "" && line.SessionID != "" {
			s.SessionID = line.SessionID
		}

		switch line.Type {
		case "assistant":
			if line.Message == nil {
				continue
			}
			var msg rawMessage
			if err := json.Unmarshal(line.Message, &msg); err != nil {
				continue
			}
			if msg.Usage.OutputTokens == 0 && msg.Usage.InputTokens == 0 {
				continue // skip empty/duplicate entries
			}

			if s.Model == "" && msg.Model != "" {
				s.Model = msg.Model
			}

			// Context size = all input tokens for this turn.
			contextSize := msg.Usage.InputTokens + msg.Usage.CacheCreationInputTokens + msg.Usage.CacheReadInputTokens

			turn := Turn{
				Index:        turnIndex,
				InputTokens:  msg.Usage.InputTokens,
				CacheCreate:  msg.Usage.CacheCreationInputTokens,
				CacheRead:    msg.Usage.CacheReadInputTokens,
				OutputTokens: msg.Usage.OutputTokens,
				ContextSize:  contextSize,
			}

			// Extract tool calls and text from content blocks.
			for _, block := range msg.Content {
				switch block.Type {
				case "tool_use":
					inputStr := string(block.Input)
					if len(inputStr) > 100 {
						inputStr = inputStr[:100] + "..."
					}
					turn.ToolCalls = append(turn.ToolCalls, ToolCall{
						Name:  block.Name,
						Input: inputStr,
					})

					// Track file reads.
					if block.Name == "Read" {
						var readInput struct {
							FilePath string `json:"file_path"`
						}
						if err := json.Unmarshal(block.Input, &readInput); err == nil && readInput.FilePath != "" {
							s.FilesRead = append(s.FilesRead, FileRead{
								Path:      readInput.FilePath,
								TurnIndex: turnIndex,
							})
						}
					}
				case "text":
					preview := block.Text
					if len(preview) > 80 {
						preview = preview[:80] + "..."
					}
					turn.TextPreview = preview
				}
			}

			// Only add turns with actual output (skip streaming intermediates).
			if msg.Usage.OutputTokens > 1 {
				s.Turns = append(s.Turns, turn)
				turnIndex++
			}

			// Track peaks.
			if contextSize > s.PeakContext {
				s.PeakContext = contextSize
			}
			s.TotalOutput += msg.Usage.OutputTokens
			if msg.Usage.CacheReadInputTokens > s.TotalCached {
				s.TotalCached = msg.Usage.CacheReadInputTokens
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan transcript: %w", err)
	}

	// Infer window size from model.
	s.WindowSize = inferWindowSize(s.Model)

	// Calculate total input from last turn.
	if len(s.Turns) > 0 {
		last := s.Turns[len(s.Turns)-1]
		s.TotalInput = last.ContextSize
	}

	return s, nil
}

func inferWindowSize(model string) int {
	switch {
	case strings.Contains(model, "claude"):
		return 200_000
	case strings.Contains(model, "gpt-4o"):
		return 128_000
	case strings.Contains(model, "gemini"):
		return 1_000_000
	default:
		return 200_000
	}
}
