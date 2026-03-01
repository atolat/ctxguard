package classifier

import (
	"testing"

	"github.com/arpan/ctxguard/internal/report"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		path string
		want report.FileCategory
	}{
		// Code
		{"src/main.go", report.CategoryCode},
		{"app.py", report.CategoryCode},
		{"index.ts", report.CategoryCode},
		{"lib.rs", report.CategoryCode},

		// Test
		{"src/main_test.go", report.CategoryTest},
		{"foo.test.js", report.CategoryTest},
		{"spec/helper_spec.rb", report.CategoryTest},
		{"__tests__/app.test.tsx", report.CategoryTest},

		// Documentation
		{"README.md", report.CategoryDocumentation},
		{"docs/guide.md", report.CategoryDocumentation},
		{"CHANGELOG.md", report.CategoryDocumentation},
		{"LICENSE", report.CategoryDocumentation},
		{"CONTRIBUTING.rst", report.CategoryDocumentation},

		// Config
		{"config.yaml", report.CategoryConfig},
		{".editorconfig", report.CategoryConfig},
		{"Makefile", report.CategoryConfig},
		{"Dockerfile", report.CategoryConfig},

		// Vendor
		{"vendor/lib/dep.js", report.CategoryVendor},
		{"node_modules/pkg/index.js", report.CategoryVendor},
		{"third_party/lib.c", report.CategoryVendor},

		// Generated
		{"api.pb.go", report.CategoryGenerated},
		{"package-lock.json", report.CategoryGenerated},
		{"go.sum", report.CategoryGenerated},
		{"yarn.lock", report.CategoryGenerated},

		// Data
		{"data.json", report.CategoryData},
		{"schema.xml", report.CategoryData},
		{"dump.csv", report.CategoryData},

		// Other
		{"logo.png", report.CategoryOther},
		{"Makefile.bak", report.CategoryOther},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := Classify(tt.path)
			if got != tt.want {
				t.Errorf("Classify(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestClassifyDeterministic(t *testing.T) {
	// The same input must always produce the same output.
	paths := []string{
		"src/main.go", "README.md", "vendor/lib/x.js",
		"config.yaml", "main_test.go", "data.json",
	}
	for _, p := range paths {
		first := Classify(p)
		for i := 0; i < 100; i++ {
			got := Classify(p)
			if got != first {
				t.Errorf("Classify(%q) not deterministic: got %q then %q", p, first, got)
			}
		}
	}
}
