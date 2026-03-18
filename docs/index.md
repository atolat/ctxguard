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

### Context Engineering (Learning Notes)
- [Context Window Anatomy](context-engineering/01-context-window-anatomy.md) — 6 layers, OS analogy, context rot

### Journal
- [The Problem](journal/01-the-problem.md) — why this exists, research, prior art

### Design
- [Architecture](design/architecture.md) — components, data flow, invariants
- [Roadmap](design/roadmap.md) — v1 CLI → v2 plugin → v3 static analysis → v4 shared server
- [Report Schema](design/report-schema.md) — canonical output format (v0.1.0)
