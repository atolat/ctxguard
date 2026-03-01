// Package walker walks a repository tree respecting .gitignore and .ctxignore.
package walker

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// IgnoreMatcher decides whether a path should be ignored.
type IgnoreMatcher struct {
	rules []ignoreRule
}

type ignoreRule struct {
	pattern  string
	negated  bool
	dirOnly  bool
	anchored bool // contains a slash (not trailing) → anchored to base
}

// NewIgnoreMatcher creates an empty matcher.
func NewIgnoreMatcher() *IgnoreMatcher {
	return &IgnoreMatcher{}
}

// LoadFile reads a gitignore-style file and appends its rules.
func (m *IgnoreMatcher) LoadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // missing ignore file is fine
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		m.addLine(line)
	}
	return scanner.Err()
}

// AddPattern adds a single ignore pattern.
func (m *IgnoreMatcher) AddPattern(pattern string) {
	m.addLine(pattern)
}

func (m *IgnoreMatcher) addLine(line string) {
	// Strip trailing whitespace (not escaped).
	line = strings.TrimRight(line, " \t")

	// Skip empty lines and comments.
	if line == "" || strings.HasPrefix(line, "#") {
		return
	}

	rule := ignoreRule{}

	// Negation.
	if strings.HasPrefix(line, "!") {
		rule.negated = true
		line = line[1:]
	}

	// Directory-only marker.
	if strings.HasSuffix(line, "/") {
		rule.dirOnly = true
		line = strings.TrimSuffix(line, "/")
	}

	// Leading slash means anchored.
	if strings.HasPrefix(line, "/") {
		rule.anchored = true
		line = line[1:]
	}

	// If the pattern contains a slash (now that we've trimmed leading/trailing),
	// it's anchored to the base directory.
	if strings.Contains(line, "/") {
		rule.anchored = true
	}

	rule.pattern = line
	m.rules = append(m.rules, rule)
}

// Match returns true if the given relative path should be ignored.
// isDir indicates whether the path is a directory.
func (m *IgnoreMatcher) Match(relPath string, isDir bool) bool {
	// Normalize to forward slashes.
	relPath = filepath.ToSlash(relPath)

	ignored := false
	for _, rule := range m.rules {
		if rule.dirOnly && !isDir {
			continue
		}

		matched := false
		if rule.anchored {
			// Match against the full relative path.
			matched = matchGlob(rule.pattern, relPath)
		} else {
			// Match against just the filename (basename).
			matched = matchGlob(rule.pattern, filepath.Base(relPath))
			// Also try matching against the full path for patterns like "dir/file".
			if !matched {
				matched = matchGlob(rule.pattern, relPath)
			}
		}

		if matched {
			ignored = !rule.negated
		}
	}
	return ignored
}

// matchGlob performs gitignore-style glob matching.
// Supports *, ?, and ** (match across directories).
func matchGlob(pattern, name string) bool {
	// Handle ** patterns.
	if strings.Contains(pattern, "**") {
		return matchDoublestar(pattern, name)
	}
	matched, _ := filepath.Match(pattern, name)
	return matched
}

// matchDoublestar handles ** which matches zero or more directories.
func matchDoublestar(pattern, name string) bool {
	parts := strings.Split(pattern, "**")
	if len(parts) == 2 {
		prefix := parts[0]
		suffix := parts[1]

		// Remove leading slash from suffix.
		suffix = strings.TrimPrefix(suffix, "/")
		// Remove trailing slash from prefix.
		prefix = strings.TrimSuffix(prefix, "/")

		if prefix == "" && suffix == "" {
			// Pattern is just "**" — matches everything.
			return true
		}
		if prefix == "" {
			// **/suffix — suffix can appear at any depth.
			if matched, _ := filepath.Match(suffix, name); matched {
				return true
			}
			// Try matching the suffix against any tail of the path.
			for i := 0; i < len(name); i++ {
				if name[i] == '/' {
					tail := name[i+1:]
					if matched, _ := filepath.Match(suffix, tail); matched {
						return true
					}
				}
			}
			return false
		}
		if suffix == "" {
			// prefix/** — matches anything under prefix.
			return strings.HasPrefix(name, prefix+"/") || name == prefix
		}
		// prefix/**/suffix.
		if !strings.HasPrefix(name, prefix+"/") {
			return false
		}
		rest := name[len(prefix)+1:]
		// suffix must match the end.
		if matched, _ := filepath.Match(suffix, filepath.Base(rest)); matched {
			return true
		}
		return matchDoublestar("**/"+suffix, rest)
	}
	// Fallback for complex patterns.
	matched, _ := filepath.Match(pattern, name)
	return matched
}
