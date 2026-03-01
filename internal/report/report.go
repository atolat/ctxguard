// Package report defines the canonical report schema for ctxguard.
// This is the single source of truth for the report format.
// Schema version: 0.1.0
package report

import "time"

// Version is the current schema version.
const Version = "0.1.0"

// Report is the top-level output of ctxguard analyze.
type Report struct {
	SchemaVersion string   `json:"schema_version"`
	ToolVersion   string   `json:"tool_version"`
	Metadata      Metadata `json:"metadata"`
	Summary       Summary  `json:"summary"`
	FileEntries   []File   `json:"file_entries"`
	Metrics       Metrics  `json:"metrics"`
	Findings      []Finding       `json:"findings,omitempty"`
	Recommendations []Recommendation `json:"recommendations,omitempty"`
}

// Metadata holds information about the analyzed repo and the run.
type Metadata struct {
	RepoPath  string    `json:"repo_path"`
	GitCommit string    `json:"git_commit,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Summary holds top-level scores. Each score is 0.0–1.0.
// Scores not yet computed are nil.
type Summary struct {
	BloatScore     *float64 `json:"bloat_score"`
	RotScore       *float64 `json:"rot_score,omitempty"`
	SecurityScore  *float64 `json:"security_score,omitempty"`
	RetrievalScore *float64 `json:"retrieval_score,omitempty"`
}

// FileCategory classifies a file's role in the repo.
type FileCategory string

const (
	CategoryCode          FileCategory = "code"
	CategoryDocumentation FileCategory = "documentation"
	CategoryTest          FileCategory = "test"
	CategoryGenerated     FileCategory = "generated"
	CategoryConfig        FileCategory = "config"
	CategoryVendor        FileCategory = "vendor"
	CategoryData          FileCategory = "data"
	CategoryOther         FileCategory = "other"
)

// File represents a single file in the report.
type File struct {
	Path     string       `json:"path"`
	Category FileCategory `json:"category"`
	Location string       `json:"location"` // e.g. "root", "docs/", "src/", ...
	SizeBytes int64       `json:"size_bytes"`
	Tokens   int64        `json:"tokens"`
	Lines    int64        `json:"lines"`
	Skipped  bool         `json:"skipped,omitempty"`
	SkipReason string     `json:"skip_reason,omitempty"`
}

// Metrics holds aggregate numbers sliced by category and directory.
type Metrics struct {
	TotalFiles    int              `json:"total_files"`
	TotalTokens   int64           `json:"total_tokens"`
	TotalBytes    int64            `json:"total_bytes"`
	TotalLines    int64            `json:"total_lines"`
	ByCategory    map[FileCategory]CategoryMetric `json:"by_category"`
	TopByTokens   []FileRef        `json:"top_by_tokens"`
}

// CategoryMetric is the aggregate for one file category.
type CategoryMetric struct {
	Files  int   `json:"files"`
	Tokens int64 `json:"tokens"`
	Bytes  int64 `json:"bytes"`
	Lines  int64 `json:"lines"`
}

// FileRef is a lightweight reference to a file with its token count,
// used in "top N" lists.
type FileRef struct {
	Path   string `json:"path"`
	Tokens int64  `json:"tokens"`
}

// Severity levels for findings.
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// FindingKind identifies the detector that produced a finding.
type FindingKind string

const (
	KindBloat    FindingKind = "bloat"
	KindRot      FindingKind = "rot"
	KindSecurity FindingKind = "security"
)

// Finding is a single issue detected by an analyzer.
type Finding struct {
	ID       string      `json:"id"`
	Severity Severity    `json:"severity"`
	Kind     FindingKind `json:"kind"`
	Message  string      `json:"message"`
	Evidence Evidence    `json:"evidence"`
}

// Evidence supports a finding with concrete data.
type Evidence struct {
	Paths    []string          `json:"paths,omitempty"`
	Snippets []string          `json:"snippets,omitempty"`
	Stats    map[string]any    `json:"stats,omitempty"`
}

// Recommendation suggests an action to reduce context bloat/rot.
type Recommendation struct {
	Action          string   `json:"action"`
	TargetPaths     []string `json:"target_paths,omitempty"`
	Rationale       string   `json:"rationale"`
	EstimatedImpact string   `json:"estimated_impact,omitempty"`
}
