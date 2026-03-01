package walker

import "testing"

func TestIgnoreMatcher(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		isDir    bool
		want     bool
	}{
		{
			name:     "simple file pattern",
			patterns: []string{"*.log"},
			path:     "debug.log",
			want:     true,
		},
		{
			name:     "simple file pattern no match",
			patterns: []string{"*.log"},
			path:     "main.go",
			want:     false,
		},
		{
			name:     "directory pattern skips dir",
			patterns: []string{"build/"},
			path:     "build",
			isDir:    true,
			want:     true,
		},
		{
			name:     "directory pattern does not match file",
			patterns: []string{"build/"},
			path:     "build",
			isDir:    false,
			want:     false,
		},
		{
			name:     "nested path",
			patterns: []string{"*.log"},
			path:     "src/debug.log",
			want:     true,
		},
		{
			name:     "anchored pattern",
			patterns: []string{"/root.txt"},
			path:     "root.txt",
			want:     true,
		},
		{
			name:     "anchored pattern no deep match",
			patterns: []string{"/root.txt"},
			path:     "sub/root.txt",
			want:     false,
		},
		{
			name:     "negation re-includes",
			patterns: []string{"*.log", "!important.log"},
			path:     "important.log",
			want:     false,
		},
		{
			name:     "negation other still ignored",
			patterns: []string{"*.log", "!important.log"},
			path:     "debug.log",
			want:     true,
		},
		{
			name:     "doublestar prefix",
			patterns: []string{"**/test.txt"},
			path:     "a/b/test.txt",
			want:     true,
		},
		{
			name:     "comment line ignored",
			patterns: []string{"# this is a comment", "*.log"},
			path:     "debug.log",
			want:     true,
		},
		{
			name:     "empty line ignored",
			patterns: []string{"", "*.log"},
			path:     "debug.log",
			want:     true,
		},
		{
			name:     "vendor directory",
			patterns: []string{"vendor/"},
			path:     "vendor",
			isDir:    true,
			want:     true,
		},
		{
			name:     "vendor subpath file - not matched by dir-only rule",
			patterns: []string{"vendor/"},
			path:     "vendor/lib/dep.js",
			isDir:    false,
			want:     false, // dir-only rule doesn't match files
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewIgnoreMatcher()
			for _, p := range tt.patterns {
				m.AddPattern(p)
			}
			got := m.Match(tt.path, tt.isDir)
			if got != tt.want {
				t.Errorf("Match(%q, isDir=%v) = %v, want %v", tt.path, tt.isDir, got, tt.want)
			}
		})
	}
}

func TestIgnoreMatcherLoadFile(t *testing.T) {
	m := NewIgnoreMatcher()
	err := m.LoadFile("testdata/nonexistent")
	if err != nil {
		t.Fatalf("LoadFile on nonexistent should not error, got: %v", err)
	}

	// Load the fakerepo's .gitignore.
	m2 := NewIgnoreMatcher()
	err = m2.LoadFile("../../testdata/fakerepo/.gitignore")
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if !m2.Match("debug.log", false) {
		t.Error("expected *.log to be ignored")
	}
	if !m2.Match("build", true) {
		t.Error("expected build/ to be ignored")
	}
}
