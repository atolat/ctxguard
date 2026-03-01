# ctxguard

Context bloat/rot monitor for AI coding agent workflows.

Analyzes a repository and produces a deterministic JSON report about token distribution, file classification, and context pollution risk.

## Quick start

```bash
go install github.com/arpan/ctxguard/cmd/ctxguard@latest

ctxguard analyze --repo .
ctxguard analyze --repo . --out report.json
```

## Docs

### Journal
- [The Problem](journal/01-the-problem.md) — why this exists, research, prior art

### Design
- [Report Schema](design/report-schema.md) — canonical output format (v0.1.0)
