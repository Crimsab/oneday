# Frequently asked questions

## Can I use my Codex subscription?

Yes. Codex is the most complete subscription-backed path in OneDay because it
can cover narrative generation and generated images without an OpenAI Platform
API key. The exact setup depends on the runtime:

- **Terminal:** install the Codex CLI, run `codex login`, and choose
  **Codex OAuth** in `oneday setup`. OneDay calls `codex exec`; the Codex CLI
  reads and refreshes its own login.
- **Docker images:** authenticate Codex on the host and enable the optional
  `imagegen-bridge` profile. Its helper copies only the host `auth.json` into a
  dedicated Docker volume. This is explicit rather than automatic, so OneDay
  never mounts your full home directory or silently imports a credential.
- **Standard Docker narrative generation:** not currently supported through the
  host Codex login. The standard gateway image does not contain the Codex CLI.
  Use LiteLLM or OpenRouter for narrative generation, or run the terminal client
  on the authenticated host.

For the Docker image profile, follow
[Optional Codex OAuth imagegen-bridge profile](docker.md#optional-codex-oauth-imagegen-bridge-profile).
The bridge bearer and Codex OAuth file are different secrets.

After creating `.env.imagegen-bridge` as described in that guide, the credential
step is:

```bash
codex login
./scripts/imagegen-bridge-copy-oauth.sh
```

## Can I use my Claude subscription?

Yes, for narrative generation through an installed and authenticated
**Claude Code** CLI. OneDay calls `claude -p`, so Claude Code remains responsible
for login and subscription access.

Enable **Claude Code** in the operator model connections and move it to the
front of the provider priority, or configure `ai.claude_code` in
`config.yaml`. Run `oneday doctor` before creating a story.

Claude Code does not provide OneDay's image-generation route and does not
provide embeddings in the current integration. Pair it with a separate image
provider and, when RAG is enabled, a separate embedding provider. The standard
Docker image also does not contain the Claude CLI, so copying the host Claude
login into that image alone is not enough.

## Does OneDay import Codex or Claude credentials automatically?

No. In a terminal installation, OneDay launches the selected local CLI and the
CLI reads its own authenticated state. OneDay does not parse or duplicate that
state.

The Docker Codex image profile is the one deliberate exception: its helper
copies only Codex `auth.json` into an isolated volume used by
`imagegen-bridge`. You must run that helper explicitly. OneDay does not support
an equivalent Claude credential import because the standard image has no Claude
runtime.

## What context does the model receive?

Every turn receives structured story state without requiring embeddings:
setting, character and world state, relevant NPCs, recent messages, branch
state, chapter information, spatial context, and the current action. This is
the normal context window and works with Codex, Claude Code, LiteLLM, and
OpenRouter.

Optional RAG adds long-term recall. OneDay periodically summarizes older story
windows, embeds those summaries and durable lore in SQLite, embeds the current
action, retrieves the closest chunks, and adds them as **Relevant Memory** to
the same request sent to the active narrative provider. Retrieval failure is
non-blocking: the turn continues with normal story context.

## Do Codex and Claude Code create embeddings?

No. OneDay's Codex and Claude Code adapters are generation adapters. They do
not expose an embeddings endpoint. RAG still works with either as the narrator
when you configure one of these embedding paths:

- local Ollama;
- a custom local HTTP embedding endpoint;
- a LiteLLM-compatible embeddings endpoint;
- OpenRouter when the selected route supports embeddings.

The narrator and embedder are independent. For example, Codex can write the
turn while local Ollama creates and searches the vectors.

## Does setup let me choose the narrator and embeddings?

Yes, with one current difference between the interfaces:

- Terminal setup offers Codex OAuth, LiteLLM, OpenRouter, and
  **Codex OAuth + local RAG embeddings**. LiteLLM and OpenRouter setup also ask
  whether to configure RAG.
- The protected browser configuration lists Codex, Claude Code, LiteLLM, and
  OpenRouter in the provider connections and priority controls. A provider only
  works when its required CLI or endpoint is reachable from the runtime.
- Claude Code is integrated in the provider router and browser configuration,
  but it does not yet have a dedicated one-click choice in the terminal setup
  wizard. It can be enabled in `config.yaml`.

Use `oneday doctor` for narrative and embedding readiness. Use
`oneday rag benchmark` for a real embedding request and dimension check.

## Can I change providers later?

Yes. Use the protected operator configuration or run
`oneday setup --reconfigure`. Provider priority is an ordered fallback chain:
disabled providers are skipped and enabled providers are tried in order.

If you change the embedding model or vector dimensions, run
`oneday rag reindex --all` so existing long-term memory uses the new vector
shape. Changing the narrator alone does not require a RAG reindex.
