// Package checkfile provides per-file context analysis for PreToolUse hooks.
package checkfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/arpan/ctxguard/internal/classifier"
	"github.com/arpan/ctxguard/internal/estimator"
	"github.com/arpan/ctxguard/internal/report"
)

// Result is the output of a file check.
type Result struct {
	Path     string             `json:"path"`
	Category report.FileCategory `json:"category"`
	Tokens   int64              `json:"tokens"`
	Lines    int64              `json:"lines"`
	SizeBytes int64             `json:"size_bytes"`
	Signal   string             `json:"signal"`   // "high", "medium", "low"
	Warning  string             `json:"warning,omitempty"`
	Advice   string             `json:"advice,omitempty"`
}

// lowSignalCategories are file categories that typically add noise to context.
var lowSignalCategories = map[report.FileCategory]bool{
	report.CategoryVendor:    true,
	report.CategoryGenerated: true,
	report.CategoryData:      true,
}

// Check analyzes a single file and returns a Result.
func Check(absPath string, repoRoot string) (*Result, error) {
	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", absPath, err)
	}

	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		relPath = absPath
	}
	relPath = filepath.ToSlash(relPath)

	cat := classifier.Classify(relPath)

	est := estimator.CharDiv4{}
	tokens, lines, err := estimator.EstimateFile(absPath, est)
	if err != nil {
		return nil, fmt.Errorf("estimate %s: %w", absPath, err)
	}

	r := &Result{
		Path:      relPath,
		Category:  cat,
		Tokens:    tokens,
		Lines:     lines,
		SizeBytes: info.Size(),
		Signal:    classifySignal(cat, tokens),
	}

	// Generate warnings for low-signal or large files.
	if lowSignalCategories[cat] {
		r.Warning = fmt.Sprintf("%s is %s (%d tokens of %s content)",
			relPath, formatTokens(tokens), tokens, cat)
	}
	if tokens > 5000 {
		if r.Warning == "" {
			r.Warning = fmt.Sprintf("%s is large (%d tokens)", relPath, tokens)
		}
	}

	// Generate advice based on category.
	switch cat {
	case report.CategoryGenerated:
		r.Advice = "Consider reading the source file instead of the generated output."
	case report.CategoryVendor:
		r.Advice = "Vendor/dependency file — typically low signal for understanding the project."
	case report.CategoryData:
		if tokens > 2000 {
			r.Advice = "Large data file — consider reading only a sample or the schema."
		}
	}

	return r, nil
}

// FormatForHook returns the result as a JSON string suitable for hook additionalContext.
func FormatForHook(r *Result) string {
	if r.Warning == "" && r.Advice == "" {
		return "" // no warning needed
	}

	parts := fmt.Sprintf("ctxguard: %s — %d tokens, category: %s, signal: %s.",
		r.Path, r.Tokens, r.Category, r.Signal)
	if r.Warning != "" {
		parts += " " + r.Warning + "."
	}
	if r.Advice != "" {
		parts += " " + r.Advice
	}
	return parts
}

// FormatJSON returns the result as JSON.
func FormatJSON(r *Result) (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func classifySignal(cat report.FileCategory, tokens int64) string {
	if lowSignalCategories[cat] {
		return "low"
	}
	if cat == report.CategoryCode || cat == report.CategoryConfig {
		return "high"
	}
	if tokens > 8000 {
		return "medium" // large files of any type get diluted
	}
	return "medium"
}

func formatTokens(n int64) string {
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
