# ctxguard

Context budget and rot monitor for AI coding agents. Gives you visibility into how much of your LLM's context window a repository consumes — and what to do about it.

## Install

```bash
go install github.com/arpan/ctxguard/cmd/ctxguard@latest
```

Or build from source:

```bash
git clone https://github.com/atolat/ctxguard.git
cd ctxguard
go build -o ctxguard ./cmd/ctxguard/
```

## Commands

### `budget` — Context window visualization

See how much of a model's context window your repo would consume.

```bash
ctxguard budget --repo .
ctxguard budget --repo . --model claude-opus-4-6
ctxguard budget --repo . --model gpt-4o
ctxguard budget --repo . --model claude-opus-4-6 --window 1000000  # 1M tier
```

### `graph` — Import dependency analysis

Find the most important files in a codebase by analyzing import relationships. Supports Go (AST), Python, JS/TS, Rust, Java, C/C++, Ruby, PHP.

```bash
ctxguard graph --repo .
ctxguard graph --repo . --top 10
ctxguard graph --repo . --json
```

### `session` — Transcript analysis

Analyze a Claude Code session transcript to see real context window usage from actual API token counts.

```bash
ctxguard session --transcript ~/.claude/projects/<project>/<session-id>.jsonl
ctxguard session --transcript <path> --window 1000000
ctxguard session --transcript <path> --json
```

### `analyze` — Full JSON report

Produces a deterministic JSON report with token distribution, file classification, bloat score, findings, and recommendations.

```bash
ctxguard analyze --repo .
ctxguard analyze --repo . --out report.json
```

### `check-file` — Per-file signal check

Check a single file's context cost and signal level. Designed for use in Claude Code PreToolUse hooks.

```bash
ctxguard check-file --path /absolute/path/to/file.go --repo .
ctxguard check-file --path /absolute/path/to/file.go --repo . --json
```

### `models` — List supported models

```bash
ctxguard models
```

## Claude Code Integration

### Quick setup

Add to your project's `CLAUDE.md`:

```markdown
## Context Budget
Run `ctxguard budget --repo .` at the start of a session to understand
this repo's context footprint.
```

### Hooks (automatic)

Add to `.claude/settings.json` for automatic context awareness:

```json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "startup",
        "hooks": [
          {
            "type": "command",
            "command": "ctxguard budget --repo ."
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "matcher": "Read",
        "hooks": [
          {
            "type": "command",
            "command": "sh -c 'INPUT=$(cat -); FILE_PATH=$(echo \"$INPUT\" | python3 -c \"import sys,json; print(json.load(sys.stdin).get(\\\"tool_input\\\",{}).get(\\\"file_path\\\",\\\"\\\"))\"); CWD=$(echo \"$INPUT\" | python3 -c \"import sys,json; print(json.load(sys.stdin).get(\\\"cwd\\\",\\\".\\\"))\"); WARN=$(ctxguard check-file --path \"$FILE_PATH\" --repo \"$CWD\" 2>/dev/null); if [ -n \"$WARN\" ]; then python3 -c \"import json; print(json.dumps({\\\"hookSpecificOutput\\\":{\\\"additionalContext\\\": $(python3 -c \"import json,sys; print(json.dumps(sys.argv[1]))\" \"$WARN\")}}))\"; fi'"
          }
        ]
      }
    ]
  }
}
```

This gives you:
- **SessionStart**: Budget summary injected into Claude's context at the start of every conversation
- **PreToolUse (Read)**: Warnings when Claude is about to read a low-signal file (vendor, generated, large data files)

### Re-inject after compaction

When Claude compresses context, the budget awareness is lost. Add a second SessionStart hook to restore it:

```json
{
  "matcher": "compact",
  "hooks": [
    {
      "type": "command",
      "command": "ctxguard budget --repo ."
    }
  ]
}
```

## What it detects

| Signal | How |
|---|---|
| Token count per file | CharDiv4 estimator (pluggable for BPE) |
| File classification | 8 categories: code, docs, test, generated, config, vendor, data, other |
| Bloat score | (vendor + generated + data tokens) / total |
| Import centrality | Dependency graph analysis — most-imported files = highest signal |
| Context growth | Real API token counts from Claude Code transcripts |
| Cache behavior | Cache hit rates from transcript analysis |

## Report schema

Output is stable JSON (`schema_version: "0.1.0"`). See `testdata/example-report.json` and [Report Schema](docs/design/report-schema.md).

## Docs

- [Architecture](docs/design/architecture.md) — components, data flow, invariants
- [Roadmap](docs/design/roadmap.md) — v1 CLI → v2 plugin → v3 static analysis → v4 shared server
- [Report Schema](docs/design/report-schema.md) — canonical output format
