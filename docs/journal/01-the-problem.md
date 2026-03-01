# 01 — The Problem

## What this is about

AI coding agents (Cursor, Claude Code, Aider, Copilot Workspace) consume repository context to do their job. That context has a budget — the context window — and everything that goes into it competes for attention.

The problem: most repos are full of stuff that actively hurts agent performance when included. Vendor dirs, stale docs, generated lock files, duplicated boilerplate, config sprawl. The agent doesn't know what matters. It ingests everything it can reach, and quality degrades.

This isn't theoretical. It's well-measured.

## The evidence

### Context length degrades performance

- **Lost in the Middle** ([Liu et al., 2023](https://arxiv.org/abs/2307.03172)) — LLMs perform significantly worse when relevant information sits in the middle of the context. Performance is U-shaped: best at the beginning and end, worst in the middle.
- **Context Rot** ([Chroma Research](https://research.trychroma.com/context-rot)) — 18 LLMs tested. Reliability drops as input length increases, even on trivial retrieval tasks. More tokens ≠ more signal.
- **NoLiMa** ([Karpinska & Iyyer, 2025](https://arxiv.org/abs/2502.05167)) — 11/13 LLMs fall below 50% accuracy at 32K tokens when you remove literal matching cues. Long context is a lie for most real tasks.
- **Context Length Alone Hurts** ([arXiv:2510.05381](https://arxiv.org/abs/2510.05381)) — 13.9%–85% performance degradation from input length alone, even with perfect retrieval. The noise itself is the problem.

### RAG doesn't save you

- **The Power of Noise** ([Cuconasu et al., 2024](https://arxiv.org/abs/2401.14887)) — Irrelevant retrieved passages don't just waste tokens. They actively mislead generation.
- **Long-Context LLMs Meet RAG** ([arXiv:2410.05983](https://arxiv.org/pdf/2410.05983)) — Adding more retrieved passages introduces noise that degrades generation quality. Retrieval saturation is real.

### Context is an attack surface

- **Indirect Prompt Injection** ([Greshake et al., 2023](https://arxiv.org/abs/2302.12173)) — The foundational paper. Instructions embedded in retrieved context can hijack agent behavior.
- **OWASP LLM Top 10** ([2025 edition](https://genai.owasp.org/llmrisk/llm01-prompt-injection/)) — Prompt injection is #1. Not #3, not #7. Number one.
- **Simon Willison's prompt injection series** ([simonwillison.net](https://simonwillison.net/series/prompt-injection/)) — Ongoing coverage from the person who named the problem. Still unsolved.

### Docs rot is real but understudied

- **Detecting Outdated Code Element References** ([arXiv:2212.01479](https://arxiv.org/abs/2212.01479)) — Studied 3,000+ GitHub repos. Most contain stale code references in docs. ICSE 2024.
- **DOCER** ([arXiv:2307.04291](https://arxiv.org/abs/2307.04291)) — Automated detection of outdated code references in READMEs and wikis.

Nobody has studied doc rot *specifically* in the context of LLM agent performance. That's the gap.

## Prior art (tools)

| Tool | What it does | What it doesn't do |
|---|---|---|
| [Repomix](https://github.com/yamadashy/repomix) | Packs a repo into one file with token counts | No classification, no bloat analysis, no security scanning |
| [scc](https://github.com/boyter/scc) | Fast line/complexity counting | Counts lines, not tokens. No context-aware classification |
| [Tokei](https://github.com/XAMPPRocky/tokei) | Code/comment/blank counting | Same — line-oriented, no LLM context lens |
| [Aider repo map](https://aider.chat/docs/repomap.html) | Graph-ranked symbol map for context selection | Tightly coupled to Aider's workflow. Not a standalone analysis tool |
| [ast-grep](https://github.com/ast-grep/ast-grep) | Structural code search via AST patterns | Search tool, not an analyzer |

None of these answer: "given this repo, how polluted is the context an AI agent would consume, and what should be excluded?"

That's what ctxguard does.

## Relevant reading

- [Effective Context Engineering for AI Agents](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents) — Anthropic's framing: find the smallest set of high-signal tokens that maximize the desired outcome.
- [Cutting Through the Noise](https://blog.jetbrains.com/research/2025/12/efficient-context-management/) — JetBrains NeurIPS 2025 workshop paper. Observation masking beats LLM summarization for agent context management. Cheaper and often better.
- [The Context Window Problem](https://factory.ai/news/context-window-problem) — Factory.ai on why naive context stuffing fails for enterprise monorepos.

## Implementation note

v1 was vibe-coded with Claude Code in a single session. The walker, classifier, estimator, and report schema all came out of one long conversation. It works. It's tested. It's also the naive version.

More deliberate, human-in-the-loop increments coming next:

- Phase 1: `ctxguard diff` for delta metrics across commits
- Phase 2: Near-duplicate detection, doc staleness heuristics
- Phase 3: Prompt injection signature scanning
- Phase 4: Retrieval dominance proxy (BM25/TF-IDF)
