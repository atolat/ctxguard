# Architecture

## Data flow

```
repo on disk
  │
  ▼
Walker ──► discovers files, respects .gitignore + .ctxignore
  │
  ▼
Classifier ──► assigns category per file (code, vendor, test, ...)
  │
  ▼
Estimator ──► computes token + line counts
  │
  ▼
Analyzer ──► aggregates metrics, computes scores, generates findings
  │
  ▼
Report (JSON to stdout or file)
```

## Components

### `internal/walker/`

Two parts: file discovery and ignore rule matching.

**Walker** (`walker.go`) — wraps `filepath.WalkDir`. Yields an `Entry` per file with path, size, and a binary flag. Always skips `.git/`. Calls the ignore matcher before yielding.

**IgnoreMatcher** (`ignore.go`) — pure Go gitignore-style pattern matcher. Supports `*`, `?`, `**`, negation (`!`), directory-only rules (`dir/`), and anchored patterns. Rules from `.gitignore` and `.ctxignore` are additive — both apply, `.ctxignore` adds exclusions on top.

Binary detection: samples the first 8KB, checks for null bytes or <90% valid UTF-8.

### `internal/classifier/`

`Classify(relPath) → FileCategory` — stateless function, no I/O. Checks rules in priority order:

1. **vendor** — path prefix (`vendor/`, `node_modules/`, `third_party/`, ...)
2. **generated** — naming patterns (`*.pb.go`, `*_generated*`, lock files)
3. **test** — path or suffix (`_test.go`, `.spec.ts`, `__tests__/`, ...)
4. **documentation** — extension (`.md`, `.rst`) or path (`docs/`) or name (`README`, `LICENSE`)
5. **config** — extension (`.yaml`, `.toml`, `.env`) or name (`Makefile`, `Dockerfile`, `go.mod`)
6. **data** — extension (`.json`, `.csv`, `.xml`, `.sql`, ...)
7. **code** — extension (`.go`, `.py`, `.ts`, `.rs`, `.c`, ~40 languages)
8. **other** — everything else

Priority order matters. A file at `vendor/lib/util_test.js` is `vendor`, not `test`.

### `internal/estimator/`

Pluggable interface:

```go
type Estimator interface {
    Estimate(content []byte) (tokens int64, lines int64)
}
```

Default implementation: `CharDiv4` — `tokens = ceil(len(content) / 4)`. Rough heuristic, good enough for relative comparisons. The interface exists so we can swap in a real BPE tokenizer later without changing anything upstream.

### `internal/analyzer/`

Orchestrator. `Run(Config) → (*Report, error)` wires everything together:

- Loads ignore files from repo root
- Walks files, classifies, estimates tokens
- Skips binary files and files exceeding `MaxFileSize` (default 1MB) — they still appear in the report with `skipped: true`
- Aggregates metrics by category
- Computes bloat score: `(vendor + generated + data tokens) / total tokens`
- Detects git commit via `git rev-parse HEAD` (only subprocess in the system)
- Generates findings (e.g., vendor >50% of tokens) and recommendations
- Sorts all output by path for determinism

### `internal/report/`

Type definitions only. No logic. Defines the JSON schema as Go structs with `json` tags.

Key types: `Report`, `Metadata`, `Summary`, `File`, `Metrics`, `Finding`, `Recommendation`.

Schema version: `0.1.0`. See [Report Schema](report-schema.md) for field details.

## Invariants

- **Deterministic**: same repo state → same report (modulo timestamp). All slices sorted by path.
- **No external dependencies**: stdlib only.
- **No LLM in the loop**: all metrics are pure computation.
- **Single subprocess**: `git rev-parse HEAD` for commit hash. Everything else is pure Go.
