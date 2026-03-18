# Roadmap

## Vision

ctxguard is a context budget and rot monitor for AI coding agents. The goal is to make context consumption **observable, predictable, and optimizable** — the same way profilers made CPU/memory usage observable for traditional software.

The tool operates on a core principle borrowed from Gastown's Zero Framework Cognition: **ctxguard provides observability, the agent provides cognition.** ctxguard never makes decisions for the agent — it surfaces data so the agent can make better decisions itself.

---

## v1 — CLI (current)

**Status: implemented**

The foundation. A Go binary that scans a repo and reports token distribution.

```
ctxguard analyze --repo .          # JSON report
ctxguard budget --repo . --model claude-opus-4-6   # visual budget
ctxguard models                    # list supported models
```

### Components

- **Walker** — file discovery, .gitignore/.ctxignore support
- **Classifier** — 8 file categories (code, docs, test, generated, config, vendor, data, other)
- **Estimator** — token counting (CharDiv4, pluggable for BPE)
- **Analyzer** — orchestrator, bloat score, findings, recommendations
- **Budget renderer** — colored terminal visualization against a model's context window

### Properties

- Deterministic: same repo → same report
- Zero external dependencies (stdlib-only Go)
- No LLM in the loop
- Single subprocess (`git rev-parse HEAD`)

---

## v2 — Claude Code Plugin

**Status: design**

Deep integration with Claude Code via the plugin system. ctxguard becomes ambient infrastructure that runs automatically.

### Distribution

```bash
# Install the binary
go install github.com/atolat/ctxguard/cmd/ctxguard@latest

# Install the Claude Code plugin
/plugin marketplace add atolat/ctxguard
/plugin install ctxguard@atolat
```

### Plugin Structure

```
ctxguard-plugin/
├── .claude-plugin/
│   └── plugin.json
├── skills/
│   └── budget/
│       └── SKILL.md              # /ctxguard:budget slash command
├── hooks/
│   └── hooks.json                # lifecycle hooks
└── settings.json
```

### Hook Integration

| Hook | Trigger | What ctxguard does |
|---|---|---|
| `SessionStart` (`startup`) | New conversation | Inject repo budget summary into context (~100 tokens) |
| `SessionStart` (`compact`) | After compaction | Re-inject budget so Claude doesn't lose awareness |
| `PreToolUse` (`Read`) | Before file read | Check file size/category, warn about bloated files |
| `PostToolUse` (`Read`) | After file read | Log what was read, track token consumption |
| `PostCompact` | Context compressed | Log compaction event for rot profiling |
| `Stop` | Session ends | Summarize session context consumption |

### PreToolUse Flow

When Claude is about to read a file, ctxguard provides signal:

```
Agent reads file → PreToolUse hook fires → ctxguard checks file
                                              │
                                    ┌─────────┴──────────┐
                                    │ Returns structured  │
                                    │ additionalContext:   │
                                    │                     │
                                    │ "12,400 tokens,     │
                                    │  category: generated,│
                                    │  bloat_score: high"  │
                                    └─────────────────────┘
                                              │
                              Claude decides whether to proceed
```

ctxguard reports facts. Claude reasons on them. The tool never blocks a read — it informs.

### CLI Additions for v2

```bash
ctxguard budget --repo . --json    # structured output for hooks
ctxguard check-file <path>         # per-file signal check (for PreToolUse)
```

### Introspection: Reading Real Agent Config

Instead of estimating overhead, read it from disk:

| Agent | System prompt | Tool config |
|---|---|---|
| Claude Code | `CLAUDE.md`, `.claude/settings.json` | `.claude/settings.json` (MCP servers) |
| Cursor | `.cursorrules`, `.cursor/rules/` | `.cursor/mcp.json` |
| Windsurf | `.windsurfrules` | Windsurf config |
| Copilot | `.github/copilot-instructions.md` | N/A |

Auto-detect which agent config files exist, count their tokens, use real numbers for overhead.

---

## v3 — Static Analysis & Session Tracking

**Status: design**

Close the semantic understanding gap without requiring an LLM.

### Import Graph Analysis

Parse imports across Go/Python/JS/TS files. Build a dependency graph. Score files by centrality.

```
pkg/auth/middleware.go  ← imported by 14 files  → HIGH SIGNAL
pkg/utils/helpers.go    ← imported by 22 files  → CRITICAL
vendor/lib/foo.go       ← imported by 0 files   → NOISE
scripts/migrate.go      ← imported by 1 file    → LOW SIGNAL
```

A file imported by many others is high-signal — an agent should prioritize it. A file with no importers is low-signal — probably safe to skip.

Implementation: Go stdlib `go/parser` for Go files. Tree-sitter for cross-language support.

### Git-Based Signals

All from `git log`, zero external deps:

- **Churn rate** — files changing most frequently are actively worked on (high signal for agents)
- **Staleness** — docs last modified months before the code they describe (rot signal)
- **Co-change patterns** — files that always change together in commits are logically coupled
- **Contributor density** — files touched by many authors may be more important (or more contested)

### Exported Surface Area

Parse AST for exported functions/types. A file with 20 exported functions is likely a core API. A file with zero exports is internal detail.

### Session Tracking

Log context consumption per session to `.ctxguard/sessions.jsonl`:

```json
{
  "timestamp": "2026-03-18T10:32:00Z",
  "model": "claude-opus-4-6",
  "session_messages": 47,
  "compactions": 1,
  "files_read": 23,
  "tokens_consumed_by_reads": 68400,
  "bloat_files_avoided": 3,
  "tokens_saved": 18200
}
```

Over time, builds a repo profile: "this repo causes compaction every ~40 messages."

### Optional LLM Pass

For deeper analysis when the user opts in:

```bash
ctxguard analyze --repo .              # fast, free, deterministic
ctxguard deep-analyze --repo .         # uses Claude API, richer
```

The deep pass can:

- Summarize each file's purpose in one line
- Detect doc-code semantic drift
- Rank files by relevance to a specific task
- Generate `.ctxignore` recommendations

---

## v4 — Shared Context Server

**Status: concept**

When multiple agents work on the same repo (subagents, or Gastown-scale swarms), they need shared awareness.

### Architecture

```
             ┌─────────────────────────┐
             │   ctxguard serve        │
             │   localhost:7700        │
             │                         │
             │   SQLite: .ctxguard/db  │
             │   ├─ file_reads         │
             │   ├─ agent_budgets      │
             │   ├─ repo_analysis      │
             │   └─ suggestions        │
             └────────┬────────────────┘
                      │ HTTP
        ┌─────────────┼─────────────────┐
        │             │                 │
   ┌────┴────┐  ┌────┴────┐     ┌─────┴────┐
   │ Claude  │  │ Subagent│     │ Subagent │
   │ (main)  │  │ Explore │     │ Plan     │
   │         │  │         │     │          │
   │ hooks:  │  │ hooks:  │     │ hooks:   │
   │ POST /  │  │ POST /  │     │ POST /   │
   │ read    │  │ read    │     │ read     │
   └─────────┘  └─────────┘     └──────────┘
```

### API Surface

```
GET  /api/budget              # repo analysis + per-agent usage
POST /api/reads               # log a file read
GET  /api/reads?file=<path>   # check if already read by another agent
GET  /api/dashboard           # swarm overview
POST /api/compact             # log compaction event
GET  /api/suggestions         # what to skip
```

### Shared Read Deduplication

Agent 1 reads `handler.go`. Agent 2 is about to read the same file. PreToolUse hook checks the server:

```json
GET /api/reads?file=pkg/auth/handler.go

{
  "already_read_by": "explore-a3f2",
  "tokens": 2100,
  "ago": "2m"
}
```

Hook injects: "this file was already read by subagent explore-a3f2 (2,100 tokens, 2 min ago). Consider using its result instead of re-reading."

### Swarm Dashboard

```
Rig: payments-refactor
────────────────────────────────────────────
Agent       Window    Used     Files  Status
────────────────────────────────────────────
Mayor       1M        120K     12     coordinating
polecat-1   200K      145K     34     ⚠ 72% full
polecat-2   200K       89K     18     working
polecat-3   200K      178K     41     ⚠ compacted 2x
polecat-4   200K       52K      8     idle
────────────────────────────────────────────
Swarm total: 1.8M tokens across 20 agents
Hot files: handler.go (read by 8 agents)
Redundant reads: ~48K tokens wasted
```

### Storage Progression

| Mode | Storage | Use case |
|---|---|---|
| CLI | None (stdout) | Quick check |
| Hooks | JSON log files | Single Claude session |
| Server | SQLite | Subagents, multi-session |
| Swarm | Dolt | Gastown-scale, shared across team |

The hook format stays the same across all modes — it switches from writing JSON files to POSTing to the server when one is running.

---

## Why Not Just Prompt Claude?

For a one-off analysis, prompting Claude directly is arguably better — Claude understands code semantics, can reason about architecture, and needs no setup.

ctxguard wins on different axes:

| | Prompting Claude | ctxguard |
|---|---|---|
| **When** | You have to remember to ask | Automatic (hooks) |
| **Token cost** | Burns context window to analyze it | Zero — runs outside the window |
| **Consistency** | Varies per conversation | Deterministic |
| **Tracking** | Gone when session ends | Persistent, trackable |
| **Pre-flight** | Claude hasn't loaded anything yet | Runs before first message |
| **Multi-agent** | Each agent analyzes independently | Shared state |

ctxguard is infrastructure, not intelligence. Like a linter isn't smarter than a senior engineer — but the linter runs on every commit and the engineer doesn't.

---

## Signal Summary

What ctxguard can report, by version:

```
                    v1 (CLI)  v2 (Plugin)  v3 (Analysis)  v4 (Server)
Token count         ✓          ✓            ✓              ✓
File classification ✓          ✓            ✓              ✓
Bloat score         ✓          ✓            ✓              ✓
Budget visualization✓          ✓            ✓              ✓
Real agent overhead            ✓            ✓              ✓
File read warnings             ✓            ✓              ✓
Compaction tracking            ✓            ✓              ✓
Import centrality                           ✓              ✓
Git churn/staleness                         ✓              ✓
Co-change coupling                          ✓              ✓
Export surface area                         ✓              ✓
Session profiling                           ✓              ✓
LLM deep analysis                           ✓              ✓
Read deduplication                                         ✓
Multi-agent dashboard                                      ✓
Swarm optimization                                         ✓
```
