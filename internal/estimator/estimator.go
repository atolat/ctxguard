// Package estimator provides pluggable token estimation for files.
package estimator

import (
	"os"
)

// Estimator computes an approximate token count from file content.
type Estimator interface {
	// Estimate returns the estimated token count and line count.
	Estimate(content []byte) (tokens int64, lines int64)
}

// CharDiv4 is the default estimator: tokens ≈ len(content) / 4.
// This is a widely-used rough heuristic for English text and code.
type CharDiv4 struct{}

// Estimate counts tokens as chars/4 and lines by counting newlines.
func (e CharDiv4) Estimate(content []byte) (tokens int64, lines int64) {
	if len(content) == 0 {
		return 0, 0
	}

	tokens = int64(len(content)+3) / 4 // ceil division

	// Count lines.
	lines = 1
	for _, b := range content {
		if b == '\n' {
			lines++
		}
	}
	return tokens, lines
}

// EstimateFile reads a file and estimates its token and line counts.
func EstimateFile(path string, est Estimator) (tokens int64, lines int64, err error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, err
	}
	tokens, lines = est.Estimate(content)
	return tokens, lines, nil
}
