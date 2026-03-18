// Package graph builds import dependency graphs from source code
// and scores files by centrality (how many other files depend on them).
package graph

import (
	"path/filepath"
	"sort"
	"strings"
)

// Node represents a file in the dependency graph.
type Node struct {
	Path       string   `json:"path"`        // relative path in repo
	Package    string   `json:"package"`      // package/module name
	Imports    []string `json:"imports"`      // packages this file imports
	ImportedBy int      `json:"imported_by"`  // number of files that import this file's package
	Centrality float64  `json:"centrality"`   // 0.0–1.0 normalized score
	Signal     string   `json:"signal"`       // "critical", "high", "medium", "low"
}

// Graph is the dependency graph for a repository.
type Graph struct {
	Nodes       []Node            `json:"nodes"`
	PackageMap  map[string][]string `json:"package_map"`  // package → files providing it
	TotalFiles  int               `json:"total_files"`
	ModulePath  string            `json:"module_path,omitempty"` // e.g. "github.com/arpan/ctxguard"
}

// Build constructs a Graph from parsed file info.
// files: map of relPath → FileInfo
func Build(files map[string]*FileInfo, modulePath string) *Graph {
	g := &Graph{
		PackageMap: make(map[string][]string),
		TotalFiles: len(files),
		ModulePath: modulePath,
	}

	// Step 1: Build package → files map and file path index.
	// fileIndex maps various ways a file can be referenced to its package.
	fileIndex := make(map[string]string) // import path → package
	for path, info := range files {
		g.PackageMap[info.Package] = append(g.PackageMap[info.Package], path)
		// Index by package name.
		fileIndex[info.Package] = info.Package
		// Index by file path (without extension) for Python-style imports.
		noExt := strings.TrimSuffix(path, filepath.Ext(path))
		fileIndex[noExt] = info.Package
		// Index by file path.
		fileIndex[path] = info.Package
		// Also index with common prefixes stripped (src/, lib/, app/).
		for _, prefix := range []string{"src/", "lib/", "app/"} {
			if strings.HasPrefix(noExt, prefix) {
				stripped := strings.TrimPrefix(noExt, prefix)
				fileIndex[stripped] = info.Package
			}
			if strings.HasPrefix(path, prefix) {
				stripped := strings.TrimPrefix(path, prefix)
				fileIndex[stripped] = info.Package
			}
		}
	}

	// Step 2: Count how many files import each package.
	importCount := make(map[string]int) // package → number of files importing it
	for _, info := range files {
		seen := make(map[string]bool)
		for _, imp := range info.Imports {
			// Resolve the import to a package via the index.
			pkg := ""
			if p, ok := fileIndex[imp]; ok {
				pkg = p
			} else {
				pkg = imp // keep as-is, might still match a package
			}
			if !seen[pkg] {
				importCount[pkg]++
				seen[pkg] = true
			}
		}
	}

	// Step 3: Score each file.
	// A file's centrality = how many other files import its package.
	maxImports := 0
	for _, count := range importCount {
		if count > maxImports {
			maxImports = count
		}
	}

	for path, info := range files {
		count := importCount[info.Package]
		centrality := 0.0
		if maxImports > 0 {
			centrality = float64(count) / float64(maxImports)
		}

		g.Nodes = append(g.Nodes, Node{
			Path:       path,
			Package:    info.Package,
			Imports:    info.Imports,
			ImportedBy: count,
			Centrality: centrality,
			Signal:     classifySignal(centrality, count),
		})
	}

	// Sort by centrality descending.
	sort.Slice(g.Nodes, func(i, j int) bool {
		return g.Nodes[i].Centrality > g.Nodes[j].Centrality
	})

	return g
}

// TopN returns the top N nodes by centrality.
func (g *Graph) TopN(n int) []Node {
	if n >= len(g.Nodes) {
		return g.Nodes
	}
	return g.Nodes[:n]
}

func classifySignal(centrality float64, importedBy int) string {
	if centrality >= 0.7 || importedBy >= 5 {
		return "critical"
	}
	if centrality >= 0.4 || importedBy >= 3 {
		return "high"
	}
	if centrality >= 0.1 || importedBy >= 1 {
		return "medium"
	}
	return "low"
}
