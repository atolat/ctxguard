package walker

import (
	"testing"
)

func TestLocation(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"README.md", "root"},
		{"src/main.go", "src/"},
		{"src/pkg/util.go", "src/"},
		{"docs/guide.md", "docs/"},
		{"a/b/c/d.txt", "a/"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := Location(tt.path)
			if got != tt.want {
				t.Errorf("Location(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestWalkFakeRepo(t *testing.T) {
	matcher := NewIgnoreMatcher()
	if err := matcher.LoadFile("../../testdata/fakerepo/.gitignore"); err != nil {
		t.Fatal(err)
	}
	if err := matcher.LoadFile("../../testdata/fakerepo/.ctxignore"); err != nil {
		t.Fatal(err)
	}

	var entries []Entry
	err := Walk("../../testdata/fakerepo", matcher, DefaultOptions(), func(e Entry) error {
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	// Should have files but vendor/ should be skipped.
	paths := make(map[string]bool)
	for _, e := range entries {
		paths[e.Path] = true
	}

	// These should be present.
	expected := []string{"README.md", "src/main.go", "src/main_test.go", "docs/guide.md", "config.yaml"}
	for _, p := range expected {
		if !paths[p] {
			t.Errorf("expected %q in walk results", p)
		}
	}

	// These should be absent (ignored).
	absent := []string{"vendor/lib/dep.js"}
	for _, p := range absent {
		if paths[p] {
			t.Errorf("did not expect %q in walk results (should be ignored)", p)
		}
	}
}
