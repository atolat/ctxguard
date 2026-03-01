package analyzer

import (
	"testing"

	"github.com/arpan/ctxguard/internal/report"
)

func TestRunFakeRepo(t *testing.T) {
	cfg := DefaultConfig("../../testdata/fakerepo")
	rpt, err := Run(cfg)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if rpt.SchemaVersion != report.Version {
		t.Errorf("schema_version = %q, want %q", rpt.SchemaVersion, report.Version)
	}

	if rpt.Metrics.TotalFiles == 0 {
		t.Error("expected at least one file")
	}

	if rpt.Metrics.TotalTokens == 0 {
		t.Error("expected non-zero total tokens")
	}

	if rpt.Summary.BloatScore == nil {
		t.Fatal("expected bloat_score to be set")
	}

	// Vendor should be excluded by .ctxignore, so no vendor files.
	if vm, ok := rpt.Metrics.ByCategory[report.CategoryVendor]; ok && vm.Files > 0 {
		t.Errorf("expected 0 vendor files (excluded by .ctxignore), got %d", vm.Files)
	}

	// Should have code files.
	if cm, ok := rpt.Metrics.ByCategory[report.CategoryCode]; !ok || cm.Files == 0 {
		t.Error("expected at least one code file")
	}

	// Should have doc files.
	if dm, ok := rpt.Metrics.ByCategory[report.CategoryDocumentation]; !ok || dm.Files == 0 {
		t.Error("expected at least one documentation file")
	}
}

func TestRunDeterministic(t *testing.T) {
	cfg := DefaultConfig("../../testdata/fakerepo")

	rpt1, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rpt2, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Metrics must be identical.
	if rpt1.Metrics.TotalFiles != rpt2.Metrics.TotalFiles {
		t.Errorf("TotalFiles not deterministic: %d vs %d", rpt1.Metrics.TotalFiles, rpt2.Metrics.TotalFiles)
	}
	if rpt1.Metrics.TotalTokens != rpt2.Metrics.TotalTokens {
		t.Errorf("TotalTokens not deterministic: %d vs %d", rpt1.Metrics.TotalTokens, rpt2.Metrics.TotalTokens)
	}
	if rpt1.Metrics.TotalBytes != rpt2.Metrics.TotalBytes {
		t.Errorf("TotalBytes not deterministic: %d vs %d", rpt1.Metrics.TotalBytes, rpt2.Metrics.TotalBytes)
	}

	// File entries must be in the same order.
	if len(rpt1.FileEntries) != len(rpt2.FileEntries) {
		t.Fatalf("FileEntries length mismatch: %d vs %d", len(rpt1.FileEntries), len(rpt2.FileEntries))
	}
	for i := range rpt1.FileEntries {
		if rpt1.FileEntries[i].Path != rpt2.FileEntries[i].Path {
			t.Errorf("FileEntries[%d] path mismatch: %q vs %q", i, rpt1.FileEntries[i].Path, rpt2.FileEntries[i].Path)
		}
		if rpt1.FileEntries[i].Tokens != rpt2.FileEntries[i].Tokens {
			t.Errorf("FileEntries[%d] tokens mismatch: %d vs %d", i, rpt1.FileEntries[i].Tokens, rpt2.FileEntries[i].Tokens)
		}
	}

	// BloatScore must be identical.
	if *rpt1.Summary.BloatScore != *rpt2.Summary.BloatScore {
		t.Errorf("BloatScore not deterministic: %f vs %f", *rpt1.Summary.BloatScore, *rpt2.Summary.BloatScore)
	}
}

func TestComputeBloatScore(t *testing.T) {
	tests := []struct {
		name       string
		total      int64
		byCategory map[report.FileCategory]report.CategoryMetric
		want       float64
	}{
		{
			name:  "zero total",
			total: 0,
			want:  0,
		},
		{
			name:  "all code",
			total: 1000,
			byCategory: map[report.FileCategory]report.CategoryMetric{
				report.CategoryCode: {Tokens: 1000},
			},
			want: 0,
		},
		{
			name:  "half vendor",
			total: 1000,
			byCategory: map[report.FileCategory]report.CategoryMetric{
				report.CategoryCode:   {Tokens: 500},
				report.CategoryVendor: {Tokens: 500},
			},
			want: 0.5,
		},
		{
			name:  "all vendor",
			total: 1000,
			byCategory: map[report.FileCategory]report.CategoryMetric{
				report.CategoryVendor: {Tokens: 1000},
			},
			want: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeBloatScore(tt.total, tt.byCategory)
			if got != tt.want {
				t.Errorf("computeBloatScore() = %f, want %f", got, tt.want)
			}
		})
	}
}
