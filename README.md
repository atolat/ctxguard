# ctxguard

Context bloat/rot monitor for AI coding agent workflows. Analyzes a repository and produces a deterministic JSON report about token distribution, file classification, and context pollution risk.

## Install

```
go install github.com/arpan/ctxguard/cmd/ctxguard@latest
```

## Usage

```
ctxguard analyze --repo .              # report to stdout
ctxguard analyze --repo . --out r.json # report to file
```

## Report schema

See `testdata/example-report.json`. Output is stable JSON (`schema_version: "0.1.0"`).

## Status

Phase 0 MVP. Walker, classifier, token estimator, bloat score. No external dependencies.
