package transcript

import (
	"fmt"
	"strings"
)

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

	bgGray    = "\033[100m"
	bgGreen   = "\033[42m"
	bgBlue    = "\033[44m"
	bgYellow  = "\033[43m"
	bgMagenta = "\033[45m"
	bgCyan    = "\033[46m"
)

// Render prints the transcript analysis to stdout.
func Render(s *Session) {
	barWidth := 60

	// Header.
	fmt.Printf("\n  %s%sSession Transcript Analysis%s\n", bold, white, reset)
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)
	fmt.Printf("  %sModel:%s  %s\n", dim, reset, s.Model)
	fmt.Printf("  %sWindow:%s %s tokens\n", dim, reset, fmtTokens(s.WindowSize))
	fmt.Printf("  %sTurns:%s  %d\n", dim, reset, len(s.Turns))
	fmt.Printf("  %sFiles:%s  %d read\n", dim, reset, len(s.FilesRead))
	fmt.Println()

	// Turn-by-turn context growth.
	fmt.Printf("  %s%sContext Growth Per Turn%s\n", bold, white, reset)
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)

	for _, t := range s.Turns {
		pct := float64(t.ContextSize) / float64(s.WindowSize) * 100
		filled := int(float64(barWidth) * float64(t.ContextSize) / float64(s.WindowSize))
		if filled < 1 && t.ContextSize > 0 {
			filled = 1
		}
		if filled > barWidth {
			filled = barWidth
		}

		// Color based on fullness.
		barColor := green
		if pct > 50 {
			barColor = yellow
		}
		if pct > 80 {
			barColor = red
		}

		bar := barColor + strings.Repeat("█", filled) + reset + dim + strings.Repeat("░", barWidth-filled) + reset

		// Tool calls annotation.
		tools := ""
		for _, tc := range t.ToolCalls {
			tools += fmt.Sprintf(" %s%s%s", cyan, tc.Name, reset)
		}

		fmt.Printf("  %sT%d%s %s %s%s (%4.1f%%)%s%s\n",
			dim, t.Index, reset,
			bar,
			white, fmtTokens(t.ContextSize), pct, reset,
			tools)
	}
	fmt.Println()

	// Context window layout at peak.
	fmt.Printf("  %s%sContext Layout at Peak (%s)%s\n", bold, white, fmtTokens(s.PeakContext), reset)
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)

	// Estimate layout segments from token counts.
	// Cached tokens ≈ system prompt + tools (stable across turns).
	// The first turn's cache_read is approximately the "base" system context.
	baseContext := 0
	messageContext := 0
	if len(s.Turns) > 0 {
		first := s.Turns[0]
		baseContext = first.CacheRead + first.CacheCreate
		if len(s.Turns) > 1 {
			last := s.Turns[len(s.Turns)-1]
			messageContext = last.ContextSize - baseContext
		}
	}

	available := s.WindowSize - s.PeakContext
	if available < 0 {
		available = 0
	}

	type layoutSeg struct {
		label  string
		tokens int
		color  string
		bg     string
		note   string
	}

	segs := []layoutSeg{
		{"System + Tools", baseContext, dim, bgGray, "cached, primacy zone"},
		{"Messages + Reads", messageContext, magenta, bgMagenta, "grows per turn"},
		{"Available", available, gray, "\033[48;5;236m", "free for generation"},
	}

	// Render horizontal bar.
	for row := 0; row < 3; row++ {
		fmt.Print("  ")
		for _, seg := range segs {
			w := barWidth * seg.tokens / s.WindowSize
			if seg.tokens > 0 && w == 0 {
				w = 1
			}
			fmt.Printf("%s%s%s", seg.bg, strings.Repeat(" ", w), reset)
		}
		fmt.Println()
	}
	fmt.Println()

	// Legend.
	for _, seg := range segs {
		pct := float64(seg.tokens) / float64(s.WindowSize) * 100
		pctStr := fmt.Sprintf("(%.1f%%)", pct)
		fmt.Printf("  %s %s%s%-18s %8s  %s%-8s %s%s\n",
			seg.bg+" "+reset,
			seg.color, reset,
			seg.label,
			fmtTokens(seg.tokens),
			dim, pctStr, seg.note, reset)
	}

	// Files read.
	if len(s.FilesRead) > 0 {
		fmt.Println()
		fmt.Printf("  %s%sFiles Loaded Into Context%s\n", bold, white, reset)
		fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)
		for _, fr := range s.FilesRead {
			path := fr.Path
			if len(path) > 50 {
				path = "…" + path[len(path)-49:]
			}
			fmt.Printf("  %sT%d%s  %s%s%s\n", dim, fr.TurnIndex, reset, cyan, path, reset)
		}
	}

	// Summary.
	fmt.Println()
	fmt.Printf("  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)

	peakPct := float64(s.PeakContext) / float64(s.WindowSize) * 100
	peakColor := green
	if peakPct > 50 {
		peakColor = yellow
	}
	if peakPct > 80 {
		peakColor = red
	}

	fmt.Printf("  %s%sPeak: %s / %s (%.1f%%)%s",
		bold, peakColor, fmtTokens(s.PeakContext), fmtTokens(s.WindowSize), peakPct, reset)
	fmt.Printf("    %sOutput: %s tokens%s\n", dim, fmtTokens(s.TotalOutput), reset)

	cachePct := float64(s.TotalCached) / float64(s.PeakContext) * 100
	if s.PeakContext > 0 {
		fmt.Printf("  %sCache hit: %.0f%% of input was cached (saves cost + latency)%s\n", dim, cachePct, reset)
	}
	fmt.Println()
}

func fmtTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
