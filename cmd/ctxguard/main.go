// ctxguard is a context bloat/rot monitor for AI coding agents.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/arpan/ctxguard/internal/analyzer"
)

const usage = `ctxguard — context bloat/rot monitor for AI coding agents

Usage:
  ctxguard analyze [flags]

Commands:
  analyze    Analyze a repository and produce a report

Flags (analyze):
  --repo <path>   Path to the repository (default: current directory)
  --out  <file>   Output file for the report (default: stdout)
  --top  <n>      Number of top-by-tokens files to include (default: 10)
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
