package graph

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Language-specific import patterns.
// Each pattern captures the imported module/path in group 1.
var importPatterns = map[string][]*regexp.Regexp{
	// Python: import foo, from foo import bar, from foo.bar import baz
	".py": {
		regexp.MustCompile(`^\s*import\s+([a-zA-Z_][\w.]*)`),
		regexp.MustCompile(`^\s*from\s+([a-zA-Z_][\w.]*)\s+import`),
	},
	// JavaScript/TypeScript: import ... from 'path', require('path')
	".js": {
		regexp.MustCompile(`(?:import\s+.*?\s+from|import)\s+['"]([^'"]+)['"]`),
		regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`),
	},
	".jsx": {
		regexp.MustCompile(`(?:import\s+.*?\s+from|import)\s+['"]([^'"]+)['"]`),
		regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`),
	},
	".ts": {
		regexp.MustCompile(`(?:import\s+.*?\s+from|import)\s+['"]([^'"]+)['"]`),
		regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`),
	},
	".tsx": {
		regexp.MustCompile(`(?:import\s+.*?\s+from|import)\s+['"]([^'"]+)['"]`),
		regexp.MustCompile(`require\s*\(\s*['"]([^'"]+)['"]\s*\)`),
	},
	// Rust: use crate::foo, mod foo
	".rs": {
		regexp.MustCompile(`^\s*use\s+(crate::[a-zA-Z_][\w:]*)`),
		regexp.MustCompile(`^\s*use\s+((?:self|super)::[a-zA-Z_][\w:]*)`),
		regexp.MustCompile(`^\s*mod\s+([a-zA-Z_]\w*)\s*;`),
	},
	// Java/Kotlin: import com.foo.bar
	".java": {
		regexp.MustCompile(`^\s*import\s+(?:static\s+)?([a-zA-Z_][\w.]*)`),
	},
	".kt": {
		regexp.MustCompile(`^\s*import\s+([a-zA-Z_][\w.]*)`),
	},
	// C/C++: #include "path" (local includes only, not <system>)
	".c": {
		regexp.MustCompile(`^\s*#include\s+"([^"]+)"`),
	},
	".h": {
		regexp.MustCompile(`^\s*#include\s+"([^"]+)"`),
	},
	".cpp": {
		regexp.MustCompile(`^\s*#include\s+"([^"]+)"`),
	},
	".hpp": {
		regexp.MustCompile(`^\s*#include\s+"([^"]+)"`),
	},
	// Ruby: require 'path', require_relative 'path'
	".rb": {
		regexp.MustCompile(`^\s*require\s+['"]([^'"]+)['"]`),
		regexp.MustCompile(`^\s*require_relative\s+['"]([^'"]+)['"]`),
	},
	// PHP: use Namespace\Class, require/include 'path'
	".php": {
		regexp.MustCompile(`^\s*use\s+([A-Za-z_\\][\w\\]*)`),
		regexp.MustCompile(`(?:require|include)(?:_once)?\s+['"]([^'"]+)['"]`),
	},
}

// ParseRepo walks a repository and extracts import relationships
// using regex-based parsing. Works across Go, Python, JS/TS, Rust,
// Java, C/C++, Ruby, PHP.
// For Go repos, falls back to AST parsing for accuracy.
func ParseRepo(repoRoot string) (map[string]*FileInfo, string, error) {
	// Check if this is a Go repo — use AST parser for precision.
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err == nil {
		return ParseGoRepo(repoRoot)
	}

	files := make(map[string]*FileInfo)

	err := filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" ||
				name == "testdata" || name == "__pycache__" || name == "venv" ||
				name == ".venv" || name == "dist" || name == "build" {
				return filepath.SkipDir
			}
			return nil
		}

		ext := filepath.Ext(path)
		patterns, ok := importPatterns[ext]
		if !ok {
			return nil // unsupported language
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		// Skip test files.
		if isTestFile(relPath) {
			return nil
		}

		imports, err := extractImports(path, patterns)
		if err != nil {
			return nil
		}

		// Resolve relative imports to repo-relative paths.
		resolvedImports := resolveImports(imports, relPath, repoRoot)

		// Use the file's directory as its "package".
		dir := filepath.Dir(relPath)
		if dir == "." {
			dir = "root"
		}

		files[relPath] = &FileInfo{
			Package: dir,
			Imports: resolvedImports,
		}
		return nil
	})

	return files, "", err
}

func extractImports(path string, patterns []*regexp.Regexp) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var imports []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, pat := range patterns {
			matches := pat.FindStringSubmatch(line)
			if len(matches) >= 2 {
				imp := matches[1]
				if !seen[imp] {
					imports = append(imports, imp)
					seen[imp] = true
				}
			}
		}
	}

	return imports, scanner.Err()
}

// resolveImports attempts to convert import paths to repo-relative paths.
// Handles: relative imports (./foo, ../foo), Python dot-path imports,
// and JS/TS path imports.
func resolveImports(imports []string, importerRelPath string, repoRoot string) []string {
	importerDir := filepath.Dir(importerRelPath)
	var resolved []string

	for _, imp := range imports {
		if strings.HasPrefix(imp, "./") || strings.HasPrefix(imp, "../") {
			// Relative import — resolve to repo path.
			joined := filepath.Join(importerDir, imp)
			joined = filepath.ToSlash(filepath.Clean(joined))
			resolved = append(resolved, joined)
		} else if strings.HasPrefix(imp, ".") && !strings.Contains(imp, "/") {
			// Python relative import: from .foo import bar → resolve against current dir
			dotImport := strings.TrimLeft(imp, ".")
			if dotImport != "" {
				joined := filepath.Join(importerDir, strings.ReplaceAll(dotImport, ".", "/"))
				resolved = append(resolved, filepath.ToSlash(joined))
			}
		} else if strings.Contains(imp, ".") && !strings.Contains(imp, "/") {
			// Python absolute import: from foo.bar.baz import X
			// Convert dots to slashes and try to match a file in the repo.
			asPath := strings.ReplaceAll(imp, ".", "/")
			resolved = append(resolved, asPath)
		} else {
			resolved = append(resolved, imp)
		}
	}
	return resolved
}

func isTestFile(relPath string) bool {
	lower := strings.ToLower(relPath)
	if strings.HasSuffix(lower, "_test.go") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.tsx") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.tsx") ||
		strings.HasSuffix(lower, "_test.py") ||
		strings.HasSuffix(lower, "_test.rs") {
		return true
	}
	if strings.Contains(lower, "/__tests__/") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/tests/") {
		return true
	}
	return false
}
