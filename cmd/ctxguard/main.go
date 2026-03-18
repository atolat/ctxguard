// ctxguard is a context bloat/rot monitor for AI coding agents.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arpan/ctxguard/internal/analyzer"
	"github.com/arpan/ctxguard/internal/budget"
	"github.com/arpan/ctxguard/internal/checkfile"
	"github.com/arpan/ctxguard/internal/graph"
)

const usage = `ctxguard — context bloat/rot monitor for AI coding agents

Usage:
  ctxguard <command> [flags]

Commands:
  analyze      Analyze a repository and produce a JSON report
  budget       Visualize context window budget for a model
  check-file   Check a single file's context cost (for hooks)
  graph        Show import dependency graph and file centrality
  models       List supported models

Flags (analyze):
  --repo <path>   Path to the repository (default: current directory)
  --out  <file>   Output file for the report (default: stdout)
  --top  <n>      Number of top-by-tokens files to include (default: 10)

Flags (budget):
  --repo    <path>    Path to the repository (default: current directory)
  --model   <model>   Model ID (default: claude-sonnet-4-6)
  --window  <tokens>  Override context window size (e.g. 1000000 for 1M)

Flags (check-file):
  --path  <file>    Absolute path to the file to check
  --repo  <path>    Path to the repository (default: current directory)
  --json            Output as JSON instead of hook format
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "analyze":
		if err := runAnalyze(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "budget":
		if err := runBudget(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "check-file":
		if err := runCheckFile(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "graph":
		if err := runGraph(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "models":
		runListModels()
	case "help", "--help", "-h":
		fmt.Print(usage)
	case "version", "--version":
		fmt.Println("ctxguard 0.1.0-dev")
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
}

func runAnalyze(args []string) error {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	repoPath := fs.String("repo", ".", "path to the repository")
	outFile := fs.String("out", "", "output file (default: stdout)")
	topN := fs.Int("top", 10, "number of top-by-tokens files")
	fs.Parse(args)

	cfg := analyzer.DefaultConfig(*repoPath)
	cfg.TopN = *topN

	rpt, err := analyzer.Run(cfg)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(rpt, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	if *outFile == "" {
		_, err = os.Stdout.Write(data)
		if err != nil {
			return err
		}
		fmt.Println() // trailing newline
		return nil
	}

	if err := os.WriteFile(*outFile, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write %s: %w", *outFile, err)
	}
	fmt.Fprintf(os.Stderr, "Report written to %s\n", *outFile)
	return nil
}

func runBudget(args []string) error {
	fs := flag.NewFlagSet("budget", flag.ExitOnError)
	repoPath := fs.String("repo", ".", "path to the repository")
	modelID := fs.String("model", "claude-sonnet-4-6", "model ID (use 'ctxguard models' to list)")
	windowOverride := fs.Int("window", 0, "override context window size in tokens (e.g. 1000000)")
	fs.Parse(args)

	model, err := budget.LookupModel(*modelID)
	if err != nil {
		return err
	}

	if *windowOverride > 0 {
		model = model.WithWindow(*windowOverride)
	}

	cfg := analyzer.DefaultConfig(*repoPath)
	rpt, err := analyzer.Run(cfg)
	if err != nil {
		return err
	}

	budget.Render(model, rpt)
	return nil
}

func runCheckFile(args []string) error {
	fs := flag.NewFlagSet("check-file", flag.ExitOnError)
	filePath := fs.String("path", "", "absolute path to the file to check")
	repoPath := fs.String("repo", ".", "path to the repository")
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	if *filePath == "" {
		return fmt.Errorf("--path is required")
	}

	absRepo, err := filepath.Abs(*repoPath)
	if err != nil {
		return err
	}

	result, err := checkfile.Check(*filePath, absRepo)
	if err != nil {
		return err
	}

	if *jsonOut {
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	// Default: hook-friendly format (only prints if there's something to warn about).
	msg := checkfile.FormatForHook(result)
	if msg != "" {
		fmt.Println(msg)
	}
	return nil
}

func runGraph(args []string) error {
	fs := flag.NewFlagSet("graph", flag.ExitOnError)
	repoPath := fs.String("repo", ".", "path to the repository")
	topN := fs.Int("top", 15, "number of top files to show")
	jsonOut := fs.Bool("json", false, "output as JSON")
	fs.Parse(args)

	absRepo, err := filepath.Abs(*repoPath)
	if err != nil {
		return err
	}

	files, modulePath, err := graph.ParseRepo(absRepo)
	if err != nil {
		return fmt.Errorf("parse repo: %w", err)
	}

	g := graph.Build(files, modulePath)

	if *jsonOut {
		data, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	graph.Render(g, *topN)
	return nil
}

func formatWindow(tokens int) string {
	if tokens >= 1_000_000 {
		return fmt.Sprintf("%.0fM", float64(tokens)/1_000_000)
	}
	return fmt.Sprintf("%dK", tokens/1000)
}

func runListModels() {
	all := budget.AllModels()
	sort.Slice(all, func(i, j int) bool {
		if all[i].Provider != all[j].Provider {
			return all[i].Provider < all[j].Provider
		}
		return all[i].ID < all[j].ID
	})

	current := ""
	for _, m := range all {
		if m.Provider != current {
			if current != "" {
				fmt.Println()
			}
			fmt.Printf("%s:\n", m.Provider)
			current = m.Provider
		}
		window := formatWindow(m.ContextWindow)
		extra := ""
		if m.MaxWindow > m.ContextWindow {
			extra = fmt.Sprintf(", up to %s", formatWindow(m.MaxWindow))
		}
		padding := strings.Repeat(" ", 24-len(m.ID))
		fmt.Printf("  %s%s%s (%s%s)\n", m.ID, padding, m.Name, window, extra)
	}
}
