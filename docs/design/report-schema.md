# Report Schema (v0.1.0)

`ctxguard analyze` outputs a single JSON object. Schema is defined in `internal/report/report.go`.

## Top-level structure

| Field | Type | Description |
|---|---|---|
| `schema_version` | string | `"0.1.0"` |
| `tool_version` | string | Build version |
| `metadata` | object | Repo path, git commit, timestamp |
| `summary` | object | Scores (0.0–1.0) |
| `file_entries` | array | Per-file data |
| `metrics` | object | Aggregates by category |
| `findings` | array | Detected issues |
| `recommendations` | array | Suggested actions |

## File categories

Files are classified by extension and path heuristics:

`code` · `documentation` · `test` · `generated` · `config` · `vendor` · `data` · `other`

## Scores

- **bloat_score** — fraction of tokens in vendor + generated + data categories
- `rot_score`, `security_score`, `retrieval_score` — reserved for future phases

## Example

See [`testdata/example-report.json`](https://github.com/atolat/ctxguard/blob/main/testdata/example-report.json).
