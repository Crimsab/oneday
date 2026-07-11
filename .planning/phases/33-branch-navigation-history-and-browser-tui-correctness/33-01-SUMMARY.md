# Summary 33.1 — Branch and history APIs

Added revision-guarded list, fork, rename, and checkout through the Go gateway
bridge and Rust routes. Timeline responses expose each branch head turn. History
and chapter endpoints are active-branch scoped, searchable, cursor-paginated,
and export complete Markdown/JSON documents with branch identity and chapters.

Branch-scoped snapshot versions, sessions, messages, and save restore lineage
prevent sibling data from leaking or disappearing after checkout and rollback.
