# Summary 16.3: Embedding Capability Selection

Embedding selection is now explicit and deterministic through `ai.embedding.provider`, with `auto` as the default safe mode. Auto mode only considers providers declared embedding-capable, while explicit selection surfaces clear reasons when the requested provider is disabled, unsupported, or missing configuration.

Config validation, example config, and TUI tests were updated so RAG no longer silently binds itself to a chat-capable but embedding-incompatible provider.
