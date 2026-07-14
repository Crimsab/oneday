# Configuration

`config.example.yaml` is the complete tracked template. Copy it to
`config.yaml`; the copy is ignored by Git and may be edited by the browser
Options workspace.

## Loading and precedence

- OneDay loads `.env` when present without overwriting variables already exported by the shell.
- `${NAME}` placeholders in `config.yaml` are expanded at runtime.
- Environment variables are the right place for API keys and bridge credentials.
- The CLI first looks for `config.yaml` in the current directory, then beside the executable.
- `oneday config show --safe` prints the effective non-secret configuration.
- `oneday setup --reconfigure` recreates the main provider/RAG choices interactively.

Never commit `config.yaml`, `.env`, database files, or the `oneday_data/` directory.

## Narrative providers

Provider order is controlled by `ai.provider_priority`. Disabled providers are
skipped; enabled providers are tried in order.

| Provider | Authentication | Notes |
| --- | --- | --- |
| `codex` | Local `codex login` | Generation only; pair it with a separate embedding provider for RAG. |
| `claude-code` | Local Claude CLI login | Shells out to the installed CLI. |
| `litellm` | `ONEDAY_LITELLM_API_KEY` | Any compatible OpenAI `/v1` endpoint; default local URL is `http://127.0.0.1:4000/v1`. |
| `openrouter` | `ONEDAY_OPENROUTER_API_KEY` | Set the model slug and enable the provider explicitly. |

Model names are deliberately user-configurable. `ai.generation.utility_model`,
`repair_model`, and `repair_fallback_models` control validation and repair calls
separately from the main narrator.

## RAG and embeddings

RAG is configured under `rag` and `ai.embedding`.

- `provider: auto` selects an enabled remote provider capable of embeddings.
- `provider: local` uses the configured Ollama or custom HTTP endpoint.
- The configured embedding dimensions must match the model output exactly.
- Changing model or dimensions requires `oneday rag reindex --all` (or
  `--story <id>`) before retrieval is reliable again.
- `oneday rag benchmark` performs a real embedding smoke test and reports latency/dimensions.

Local Ollama defaults to `http://127.0.0.1:11434` and `bge-m3`. In Docker,
replace the host with `host.docker.internal` when Ollama runs outside the container.

## Visual generation

Visual settings live under `ai.image_generation` and can be overridden with
`ONEDAY_IMAGEGEN_*` variables.

- General scene and character design defaults to `openai/gpt-image-2`.
- Transparent map symbols use the separate `map_icon_model`, defaulting to
  `openai/gpt-image-1`.
- `openclaw-bridge` calls `openclaw_bridge_url`; an OpenAI-compatible provider
  uses `base_url` and `api_key` instead.
- `auto_generate` controls background generation; failed work remains non-blocking.
- `append_negative_prompt` appends the saved negative direction to provider requests.
- Size, aspect ratio, resolution, output format, background, and timeout can be
  set globally or separately for locations and characters.

The browser story wizard offers Auto, Photorealistic, Cinematic Fantasy,
Illustrated Fantasy, Anime, and Custom visual styles. The style prompt is saved
with the story so later assets remain consistent.

## Game and storage

- `data_dir` contains `oneday.db`, generated visual/audio assets, and story data.
- `game.autosave_every` controls automatic save frequency.
- `game.typewriter_effect` and `typewriter_speed` affect terminal presentation.
- `game.visible_private_thoughts` is a storyteller/debug option and is off by default.
- `game.reward_budget` accepts `generous`, `balanced`, or `harsh`.

SQLite is canonical for both clients. Back up the entire data directory while
OneDay is stopped, or use a SQLite-safe backup procedure.
