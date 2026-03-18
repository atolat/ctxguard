package graph

import (
	"fmt"
	"strings"
)

// ANSI colors.
const (
	reset   = "\033[0m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	white   = "\033[37m"
	gray    = "\033[90m"
)

var signalColors = map[string]string{
	"critical": red,
	"high":     yellow,
	"medium":   blue,
	"low":      gray,
}

// Render prints the dependency graph analysis to stdout.
func Render(g *Graph, topN int) {
	fmt.Printf("\n  %s%sDependency Graph: %d files analyzed%s\n", bold, white, g.TotalFiles, reset)
	if g.ModulePath != "" {
		fmt.Printf("  %s%s%s\n", dim, g.ModulePath, reset)
	}
	fmt.Printf("  %s%s%s\n\n", dim, strings.Repeat("─", 60), reset)

	nodes := g.TopN(topN)
	if len(nodes) == 0 {
		fmt.Printf("  %sNo import relationships found.%s\n\n", dim, reset)
		return
	}

	// Find max imported_by for bar scaling.
	maxImports := 0
	for _, n := range nodes {
		if n.ImportedBy > maxImports {
			maxImports = n.ImportedBy
		}
	}

	barWidth := 30
	fmt.Printf("  %-35s %s  %s\n", "File", "Imported by", "Signal")
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", 60), reset)

	for _, n := range nodes {
		color := signalColors[n.Signal]
		bar := ""
		if maxImports > 0 {
			filled := barWidth * n.ImportedBy / maxImports
			if n.ImportedBy > 0 && filled == 0 {
				filled = 1
			}
			bar = color + strings.Repeat("█", filled) + reset + strings.Repeat(" ", barWidth-filled)
		}

		path := n.Path
		if len(path) > 33 {
			path = "…" + path[len(path)-32:]
		}

		fmt.Printf("  %-35s %s %2d  %s%s%s\n",
			path, bar, n.ImportedBy, color, n.Signal, reset)
	}

	// Summary stats.
	fmt.Printf("\n  %s%s%s\n", dim, strings.Repeat("─", 60), reset)

	critical, high, medium, low := 0, 0, 0, 0
	for _, n := range g.Nodes {
		switch n.Signal {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}

	fmt.Printf("  %s●%s Critical: %d  ", red, reset, critical)
	fmt.Printf("%s●%s High: %d  ", yellow, reset, high)
	fmt.Printf("%s●%s Medium: %d  ", blue, reset, medium)
	fmt.Printf("%s●%s Low: %d\n\n", gray, reset, low)
}
