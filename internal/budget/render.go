package budget

import (
	"fmt"
	"strings"

	"github.com/arpan/ctxguard/internal/report"
)

// ANSI color codes.
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

	bgRed     = "\033[41m"
	bgGreen   = "\033[42m"
	bgYellow  = "\033[43m"
	bgBlue    = "\033[44m"
	bgMagenta = "\033[45m"
	bgCyan    = "\033[46m"
	bgGray    = "\033[100m"
)

// categoryDisplay maps file categories to display properties.
type categoryDisplay struct {
	fg    string
	bg    string
	label string
}

var catDisplay = map[report.FileCategory]categoryDisplay{
	report.CategoryCode:          {fg: green, bg: bgGreen, label: "Code"},
	report.CategoryDocumentation: {fg: blue, bg: bgBlue, label: "Docs"},
	report.CategoryConfig:        {fg: cyan, bg: bgCyan, label: "Config"},
	report.CategoryTest:          {fg: magenta, bg: bgMagenta, label: "Tests"},
	report.CategoryData:          {fg: yellow, bg: bgYellow, label: "Data"},
	report.CategoryGenerated:     {fg: yellow, bg: bgYellow, label: "Generated"},
	report.CategoryVendor:        {fg: red, bg: bgRed, label: "Vendor"},
	report.CategoryOther:         {fg: dim, bg: bgGray, label: "Other"},
}

// categoryOrder defines render order.
var categoryOrder = []report.FileCategory{
	report.CategoryCode,
	report.CategoryDocumentation,
	report.CategoryConfig,
	report.CategoryTest,
	report.CategoryData,
	report.CategoryGenerated,
	report.CategoryVendor,
	report.CategoryOther,
}

// agentOverhead estimates tokens consumed by non-repo context layers.
type agentOverhead struct {
	SystemPrompt       int
	ConversationBuffer int
	ToolSchemas        int
	GenerationHeadroom int
}

func defaultOverhead() agentOverhead {
	return agentOverhead{
		SystemPrompt:       2_000,
		ConversationBuffer: 10_000,
		ToolSchemas:        3_000,
		GenerationHeadroom: 4_000,
	}
}

func (o agentOverhead) total() int {
	return o.SystemPrompt + o.ConversationBuffer + o.ToolSchemas + o.GenerationHeadroom
}

// segment represents one colored chunk of the bar.
type segment struct {
	label  string
	tokens int
	fg     string
	bg     string
}

// Render prints a colored context window budget to stdout.
func Render(model Model, rpt *report.Report) {
	overhead := defaultOverhead()
	window := model.ContextWindow
	repoTokens := int(rpt.Metrics.TotalTokens)
	overheadTokens := overhead.total()
	usedTokens := repoTokens + overheadTokens
	availableTokens := window - usedTokens
	if availableTokens < 0 {
		availableTokens = 0
	}

	barWidth := 60

	// Header.
	fmt.Printf("\n  %s%s%s %s%s  %s\n", bold, white, model.Name, dim, formatTokens(window)+" context window", reset)
	fmt.Printf("  %s%s%s\n\n", dim, strings.Repeat("─", barWidth+2), reset)

	// Build segments: overhead, then repo categories, then available.
	var segs []segment

	segs = append(segs, segment{"Overhead", overheadTokens, dim, bgGray})

	for _, cat := range categoryOrder {
		cm, ok := rpt.Metrics.ByCategory[cat]
		if !ok || cm.Tokens == 0 {
			continue
		}
		d := catDisplay[cat]
		segs = append(segs, segment{d.label, int(cm.Tokens), d.fg, d.bg})
	}

	segs = append(segs, segment{"Available", availableTokens, gray, "\033[48;5;236m"})

	// Render the main horizontal bar (3 rows tall for visibility).
	for row := 0; row < 3; row++ {
		fmt.Print("  ")
		for _, s := range segs {
			w := barWidth * s.tokens / window
			if s.tokens > 0 && w == 0 {
				w = 1
			}
			fmt.Printf("%s%s%s", s.bg, strings.Repeat(" ", w), reset)
		}
		fmt.Println()
	}
	fmt.Println()

	// Legend: each segment with its color swatch, label, tokens, and percentage.
	for _, s := range segs {
		pct := float64(s.tokens) / float64(window) * 100
		warning := ""
		if s.label == "Vendor" && pct > 5 {
			warning = fmt.Sprintf("  %s⚠ bloat%s", red, reset)
		}
		fmt.Printf("  %s  %s %-12s %8s  %s(%4.1f%%)%s%s\n",
			s.bg+" "+reset,
			s.fg, s.label, formatTokens(s.tokens),
			dim, pct, reset, warning)
	}

	// Summary line.
	fmt.Printf("\n  %s%s%s\n", dim, strings.Repeat("─", barWidth+2), reset)

	availColor := green
	availPct := float64(availableTokens) / float64(window) * 100
	if availPct < 50 {
		availColor = yellow
	}
	if availPct < 20 {
		availColor = red
	}

	fmt.Printf("  %s%sUsed: %s / %s (%.1f%%)%s",
		bold, white,
		formatTokens(usedTokens), formatTokens(window),
		float64(usedTokens)/float64(window)*100,
		reset)
	fmt.Printf("    %s%sFree: %s (%.1f%%)%s\n\n",
		bold, availColor,
		formatTokens(availableTokens), availPct,
		reset)

	// Overflow warning.
	if availableTokens == 0 {
		overflow := usedTokens - window
		fmt.Printf("  %s%s ⚠  OVERFLOW: %s tokens over budget %s\n\n", bold, bgRed, formatTokens(overflow), reset)
	}
}

func formatTokens(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
