# Summary 12.3: Command Routing, Provider Decoupling, and Save Hygiene

Registered `/craft` and `/crafting` in the command parser, fixed autosave replacement so it removes superseded snapshot files from disk, selected the first enabled embedding-capable provider instead of hard-binding RAG to LiteLLM, and corrected combat/crafting prompt scaffolding to use proper `system` messages. Added focused regression tests around rollback restore, auxiliary history writes, chat-message constraints, command parsing, and embedding-provider selection.
