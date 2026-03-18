package graph

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo holds parsed import data for a single source file.
type FileInfo struct {
	Package string   // package name (fully qualified for Go)
	Imports []string // imported packages
}

// ParseGoRepo walks a Go repository and extracts import relationships.
// modulePath is read from go.mod (e.g. "github.com/arpan/ctxguard").
// Returns map of relPath → FileInfo.
func ParseGoRepo(repoRoot string) (map[string]*FileInfo, string, error) {
	modulePath, err := readModulePath(repoRoot)
	if err != nil {
		modulePath = "" // not a Go module, still try to parse
	}

	files := make(map[string]*FileInfo)
	fset := token.NewFileSet()

	err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}

		// Skip hidden directories, vendor, testdata.
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") || name == "vendor" || name == "node_modules" || name == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only parse .go files (skip tests for import graph).
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		relPath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return nil
		}
		relPath = filepath.ToSlash(relPath)

		fi, err := parseGoFile(fset, path, relPath, modulePath, repoRoot)
		if err != nil {
			return nil // skip unparseable files
		}

		files[relPath] = fi
		return nil
	})

	return files, modulePath, err
}

func parseGoFile(fset *token.FileSet, absPath, relPath, modulePath, repoRoot string) (*FileInfo, error) {
	f, err := parser.ParseFile(fset, absPath, nil, parser.ImportsOnly)
	if err != nil {
		return nil, err
	}

	// Build the fully qualified package path.
	dir := filepath.Dir(relPath)
	pkgPath := dir
	if modulePath != "" && dir != "." {
		pkgPath = modulePath + "/" + dir
	} else if modulePath != "" {
		pkgPath = modulePath
	}

	var imports []string
	for _, imp := range f.Imports {
		impPath := strings.Trim(imp.Path.Value, `"`)

		// Only track internal imports (within the module).
		if modulePath != "" && strings.HasPrefix(impPath, modulePath) {
			imports = append(imports, impPath)
		}
	}

	return &FileInfo{
		Package: pkgPath,
		Imports: imports,
	}, nil
}

// readModulePath extracts the module path from go.mod.
func readModulePath(repoRoot string) (string, error) {
	modPath := filepath.Join(repoRoot, "go.mod")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", nil
}

// ParseGoFileAST is exported for potential reuse — parses a single Go file
// and returns its package clause and import list.
func ParseGoFileAST(path string) (pkgName string, imports []string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	if err != nil {
		return "", nil, err
	}

	pkgName = ""
	if f.Name != nil {
		pkgName = f.Name.Name
	}

	for _, imp := range f.Imports {
		imports = append(imports, strings.Trim(imp.Path.Value, `"`))
	}

	// Ignore unused function warning.
	_ = ast.Print
	return pkgName, imports, nil
}
