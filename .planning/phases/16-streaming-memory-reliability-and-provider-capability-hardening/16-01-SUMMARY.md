# Summary 16.1: Unified Structured Request Path

Sync and streaming OpenAI-compatible calls now share the same request-construction path for model selection, response-format selection, plugin injection, and provider hints.

Streaming requests now retry without `response_format` when structured output is rejected, and can fall back to a one-shot complete call when a provider rejects streaming entirely. Focused provider tests now cover structured stream parity, structured fallback, and complete-as-stream fallback behavior.
