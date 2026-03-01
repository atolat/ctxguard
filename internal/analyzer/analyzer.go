// Package analyzer orchestrates file walking, classification, and token
// estimation to produce a report.
package analyzer

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/arpan/ctxguard/internal/classifier"
	"github.com/arpan/ctxguard/internal/estimator"
	"github.com/arpan/ctxguard/internal/report"
	"github.com/arpan/ctxguard/internal/walker"
)

// Config holds analyzer settings.
type Config struct {
	RepoPath    string
	TopN        int // number of top-by-tokens files to include (default 10)
	MaxFileSize int64
}

// DefaultConfig returns defaults.
func DefaultConfig(repoPath string) Config {
	return Config{
		RepoPath:    repoPath,
		TopN:        10,
		MaxFileSize: 1 << 20, // 1 MB
	}
}

// Run performs the full analysis and returns a Report.
func Run(cfg Config) (*report.Report, error) {
	absPath, err := filepath.Abs(cfg.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("resolve repo path: %w", err)
	}

	// Load ignore rules.
	matcher := walker.NewIgnoreMatcher()
	if err := matcher.LoadFile(filepath.Join(absPath, ".gitignore")); err != nil {
		return nil, fmt.Errorf("load .gitignore: %w", err)
	}
	if err := matcher.LoadFile(filepath.Join(absPath, ".ctxignore")); err != nil {
		return nil, fmt.Errorf("load .ctxignore: %w", err)
	}

	est := estimator.CharDiv4{}
	opts := walker.DefaultOptions()
	opts.MaxFileSize = cfg.MaxFileSize

	var files []report.File
	byCategory := make(map[report.FileCategory]*report.CategoryMetric)

	var totalTokens, totalBytes, totalLines int64

	err = walker.Walk(absPath, matcher, opts, func(entry walker.Entry) error {
		cat := classifier.Classify(entry.Path)
		loc := walker.Location(entry.Path)

		f := report.File{
			Path:      entry.Path,
			Category:  cat,
			Location:  loc,
			SizeBytes: entry.Size,
		}

		if entry.IsBinary {
			f.Skipped = true
			f.SkipReason = "binary"
		} else if entry.Size > opts.MaxFileSize {
			f.Skipped = true
			f.SkipReason = fmt.Sprintf("exceeds max size (%d bytes)", opts.MaxFileSize)
		} else {
			tokens, lines, err := estimator.EstimateFile(entry.AbsPath, est)
			if err != nil {
				return fmt.Errorf("estimate %s: %w", entry.Path, err)
			}
			f.Tokens = tokens
			f.Lines = lines
		}

		files = append(files, f)

		// Update category metrics.
		cm, ok := byCategory[cat]
		if !ok {
			cm = &report.CategoryMetric{}
			byCategory[cat] = cm
		}
		cm.Files++
		cm.Tokens += f.Tokens
		cm.Bytes += f.SizeBytes
		cm.Lines += f.Lines

		totalTokens += f.Tokens
		totalBytes += f.SizeBytes
		totalLines += f.Lines

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk: %w", err)
	}

	// Sort files by path for deterministic output.
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Build top-by-tokens list.
	topN := cfg.TopN
	if topN <= 0 {
		topN = 10
	}
	sorted := make([]report.File, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Tokens > sorted[j].Tokens
	})
	if len(sorted) > topN {
		sorted = sorted[:topN]
	}
	topByTokens := make([]report.FileRef, 0, len(sorted))
	for _, f := range sorted {
		if f.Tokens > 0 {
			topByTokens = append(topByTokens, report.FileRef{
				Path:   f.Path,
				Tokens: f.Tokens,
			})
		}
	}

	// Convert byCategory to value map.
	byCategoryVal := make(map[report.FileCategory]report.CategoryMetric, len(byCategory))
	for k, v := range byCategory {
		byCategoryVal[k] = *v
	}

	// Compute bloat score.
	bloatScore := computeBloatScore(totalTokens, byCategoryVal)

	// Detect git commit.
	gitCommit := detectGitCommit(absPath)

	// Build findings.
	findings := generateFindings(totalTokens, byCategoryVal)

	// Build recommendations.
	recommendations := generateRecommendations(totalTokens, byCategoryVal)

	rpt := &report.Report{
		SchemaVersion: report.Version,
		ToolVersion:   "0.1.0-dev",
		Metadata: report.Metadata{
			RepoPath:  absPath,
			GitCommit: gitCommit,
			Timestamp: time.Now().UTC(),
		},
		Summary: report.Summary{
			BloatScore: &bloatScore,
		},
		FileEntries: files,
		Metrics: report.Metrics{
			TotalFiles:  len(files),
			TotalTokens: totalTokens,
			TotalBytes:  totalBytes,
			TotalLines:  totalLines,
			ByCategory:  byCategoryVal,
			TopByTokens: topByTokens,
		},
		Findings:        findings,
		Recommendations: recommendations,
	}

	return rpt, nil
}

// computeBloatScore returns a 0–1 score estimating how "bloated" the context is.
// Heuristic: fraction of tokens in non-code categories (vendor, generated, data, docs).
func computeBloatScore(totalTokens int64, byCategory map[report.FileCategory]report.CategoryMetric) float64 {
	if totalTokens == 0 {
		return 0
	}

	nonCodeTokens := int64(0)
	bloatCategories := []report.FileCategory{
		report.CategoryVendor,
		report.CategoryGenerated,
		report.CategoryData,
	}
	for _, cat := range bloatCategories {
		if m, ok := byCategory[cat]; ok {
			nonCodeTokens += m.Tokens
		}
	}

	score := float64(nonCodeTokens) / float64(totalTokens)
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// detectGitCommit tries to get the HEAD commit hash.
func detectGitCommit(repoPath string) string {
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// generateFindings produces bloat findings based on token distribution.
func generateFindings(totalTokens int64, byCategory map[report.FileCategory]report.CategoryMetric) []report.Finding {
	if totalTokens == 0 {
		return nil
	}

	var findings []report.Finding

	// Check vendor dominance.
	if vendor, ok := byCategory[report.CategoryVendor]; ok {
		pct := float64(vendor.Tokens) / float64(totalTokens) * 100
		if pct > 50 {
			findings = append(findings, report.Finding{
				ID:       "BLOAT-001",
				Severity: report.SeverityWarning,
				Kind:     report.KindBloat,
				Message:  fmt.Sprintf("Vendor directory contains %.1f%% of total tokens", pct),
				Evidence: report.Evidence{
					Paths: []string{"vendor/"},
					Stats: map[string]any{"vendor_token_pct": pct},
				},
			})
		}
	}

	// Check generated file dominance.
	if gen, ok := byCategory[report.CategoryGenerated]; ok {
		pct := float64(gen.Tokens) / float64(totalTokens) * 100
		if pct > 30 {
			findings = append(findings, report.Finding{
				ID:       "BLOAT-002",
				Severity: report.SeverityInfo,
				Kind:     report.KindBloat,
				Message:  fmt.Sprintf("Generated files contain %.1f%% of total tokens", pct),
				Evidence: report.Evidence{
					Stats: map[string]any{"generated_token_pct": pct},
				},
			})
		}
	}

	return findings
}

// generateRecommendations suggests actions based on findings.
func generateRecommendations(totalTokens int64, byCategory map[report.FileCategory]report.CategoryMetric) []report.Recommendation {
	if totalTokens == 0 {
		return nil
	}

	var recs []report.Recommendation

	if vendor, ok := byCategory[report.CategoryVendor]; ok {
		pct := float64(vendor.Tokens) / float64(totalTokens) * 100
		if pct > 50 {
			recs = append(recs, report.Recommendation{
				Action:          "Add vendor/ to .ctxignore",
				TargetPaths:     []string{"vendor/"},
				Rationale:       fmt.Sprintf("Vendor files dominate context at %.1f%%; excluding them reduces token count significantly", pct),
				EstimatedImpact: fmt.Sprintf("~%d tokens saved", vendor.Tokens),
			})
		}
	}

	return recs
}
