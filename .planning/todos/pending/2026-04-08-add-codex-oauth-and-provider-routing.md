---
created: 2026-04-08T20:13:48+02:00
title: Add Codex OAuth and provider routing
area: general
files:
  - internal/ai/
  - internal/config/
  - docs/
---

## Problem

The current AI routing is centered around Claude Code CLI plus proxy-based providers.
There is interest in expanding the provider story so the project can support Codex-native
authentication/routing as an additional first-class option, while also keeping practical
paths open for OpenRouter/LiteLLM-backed models such as Grok 4.1.

Right now this idea only exists in conversation context and could easily get lost after
the Phase 8 close-out.

## Solution

Plan a follow-up phase or todo that evaluates:

1. whether Codex OAuth can be integrated cleanly as a provider/auth source without
   overfitting the runtime to one CLI
2. how that coexists with existing provider routing and fallback logic
3. whether the simpler near-term path is to keep Codex out of runtime auth and use
   Grok 4.1 via OpenRouter/LiteLLM as the practical additional model route

Expected touchpoints:
- provider interfaces and adapters in `internal/ai/`
- config surface in `internal/config/`
- user-facing setup docs
