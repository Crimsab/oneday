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
- `oneday config locale en|it|auto` saves the terminal interface language;
  `auto` follows `LC_ALL`, `LC_MESSAGES`, then `LANG`.
- `oneday setup --reconfigure` recreates the main provider/RAG choices interactively.

Never commit `config.yaml`, `.env`, database files, or the `oneday_data/` directory.

## Profiles and local files

The terminal and a standalone desktop profile are separate installations. The
terminal resolves `config.yaml` from the current directory and then beside the
executable; its `data_dir` is controlled by that configuration. The default
relative `./oneday_data` therefore belongs to the directory from which that
configuration is resolved. Copy or back up the full configured data directory,
not just `oneday.db`, because it also contains generated assets and related
story files.

Desktop remote mode stores only the selected server setting locally. It does not
create a local `data_dir`. Desktop standalone mode creates an opaque, isolated
profile with its own absolute `data_dir`; it does not read or merge the terminal
configuration. The usual desktop locations are documented in [Desktop](desktop.md).
Changing between remote and standalone is not synchronization.

## Gateway authentication and reverse proxies

The gateway listens on `127.0.0.1:8788` by default. For an interactive local
start it may print a one-shot bootstrap URL; treat that URL as a credential and
do not paste it into tickets, screenshots, shell history, or bookmarks. A
non-interactive process or a gateway bound beyond loopback must be supplied a
bootstrap token (`ONEDAY_GATEWAY_BOOTSTRAP_TOKEN`) unless it is intentionally
configured for direct bearer access only.

`ONEDAY_GATEWAY_AUTH_TOKEN` is the direct bearer credential. It is distinct
from the bootstrap token and is not a browser-login substitute. Both values
must be at least 32 bytes and must remain in a secret store or process
environment, never in `config.yaml` or a URL. Browser bootstrap sessions are
signed and expire after 12 hours. Interactive bootstrap credentials are
consumed once. An explicitly configured bootstrap credential remains available
for reauthentication, while issued sessions remain valid across gateway
restarts as long as that configured credential does not change.

When serving a public origin through a reverse proxy:

- terminate TLS at the public origin and preserve the browser's `Host` value;
- add that host (including its port when non-default) to
  `ONEDAY_GATEWAY_ALLOWED_HOSTS` so gateway Host validation can accept it;
- avoid rewriting the application under a path prefix—the desktop client and
  gateway expect an origin, not an additional path;
- keep the gateway on a private/loopback network where practical, and have the
  proxy provide any required remote authentication and rate limiting.

Do not expose a bare HTTP listener or rely on a reverse proxy to repair an
incorrect Host/Origin configuration. See the [security threat model](security-threat-model.md)
for the boundary and residual risks.

## Player preferences and operator configuration

The browser keeps player preferences separate from installation configuration.
**Player preferences** contain only local presentation and play choices:
appearance, typography, accessibility behavior, spoken-audio preferences,
story visual direction, message detail visibility, and preference/theme
import-export. They are stored in the browser and can be reset or moved without
changing the server.

**Operator configuration** is the authenticated gateway area for provider and
model routing, endpoints, write-only credentials, runtime readiness, reload,
and redacted support diagnostics. It uses the existing `/api/config/models` and
`/api/setup/readiness` contract in both standalone and remote modes; it does
not create a second configuration store or backend. Saved credentials are never
returned to the browser—only their configured status is exposed.

Story onboarding creates a story. It may link an operator to the protected
configuration area when the installation is not ready, but it does not turn
provider setup into a player preference.

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

- `codex-oauth` is the first/default provider and calls the Codex-only
  [`imagegen-bridge`](https://github.com/Crimsab/imagegen-bridge) native
  `/v1/images` contract.
- The default route is `codex-responses` with
  `codex-app-server:gpt-image-2` as the ordered technical-error fallback. Both
  use Codex/ChatGPT OAuth; neither consumes `OPENAI_API_KEY`.
- `imagegen_bridge_provider` and `imagegen_bridge_map_icon_provider` accept only
  `codex-responses` or `codex-app-server`. Provider and model are explicit;
  `provider`/`model` and `map_icon_provider`/`map_icon_model` may differ.
- `imagegen_bridge_fallbacks` accepts ordered `PROVIDER` or `PROVIDER:MODEL`
  entries. `imagegen_bridge_fallback_policy` is `on_unavailable` or `on_error`;
  `imagegen_bridge_compatibility` is `strict`, `normalize`, or `best_effort`.
- The bridge bearer token belongs in `ONEDAY_IMAGEGEN_BRIDGE_TOKEN`, not in
  browser state or committed YAML. The Options panel reports only whether it is
  configured.
- Direct adapters are available for OpenAI Platform, compatible/LiteLLM image
  endpoints, Gemini, fal.ai, Replicate, Stability, and Azure OpenAI. Their
  endpoints, API keys, versions, and configured models live under
  `ai.image_generation.providers`; keys are write-only through Settings.
- Legacy `openclaw-bridge` still calls `openclaw_bridge_url`.
- `auto_generate` controls background generation; failed work remains non-blocking.
- `append_negative_prompt` enables saved negative direction. The native bridge
  receives it as `negative_prompt`; legacy adapters receive a merged prompt.
- Size, aspect ratio, resolution, output format, background, and timeout can be
  set globally or separately for locations and characters.

The browser story wizard offers Auto, Photorealistic, Cinematic Fantasy,
Illustrated Fantasy, Anime, and Custom visual styles. The style prompt is saved
with the story so later assets remain consistent.

The tracked public template sets `auto_generate: false`. Turn it on only after
the selected bridge or compatible endpoint is reachable; otherwise OneDay will
correctly preserve text turns but accumulate avoidable failed background jobs.

## Spoken audio

Speech settings live under `ai.tts`. OneDay implements two adapters:

- `cloud` calls an OpenAI-compatible `/audio/speech` endpoint. Put its key in
  `ONEDAY_TTS_API_KEY` and reference that variable from `config.yaml`.
- `local` expects a Piper-compatible service with `/voices` and `/synthesize`.

`provider_order` controls fallback, while each endpoint has independent
`enabled`, `base_url`, `model`, `voice`, `version`, and `languages` fields.
Generated audio is stored below `output_dir` and cached against committed text
and voice settings. Both providers are disabled in the public template; text
remains available when TTS is off or a job fails. See [Generated media](media.md)
for complete examples and Docker-host addressing.

## Optional observability export

OneDay always keeps redacted generation lineage and failure state in SQLite.
It can additionally export runtime and image-generation spans to any
OTLP/HTTP-protobuf backend without changing the application code.

- Set `OTEL_EXPORTER_OTLP_ENDPOINT` to a collector base endpoint; the exporter
  appends `/v1/traces`. Alternatively, set the signal-specific
  `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to the complete traces URL. Export is
  disabled when neither endpoint exists.
- `ONEDAY_OTEL_ENABLED=false` is an explicit kill switch. Set it to `true` to
  require exporter initialization and fail startup on invalid exporter config.
- Standard `OTEL_EXPORTER_OTLP_HEADERS`, `OTEL_TRACES_SAMPLER`,
  `OTEL_TRACES_SAMPLER_ARG`, `OTEL_SERVICE_NAME`, and
  `OTEL_RESOURCE_ATTRIBUTES` variables are supported. Headers and endpoints are
  never returned by `/api/health` or written to OneDay's telemetry tables.
- Image spans include job/asset kind, requested and resolved model, actual
  provider, duration, and a bounded error class. Prompts, revised prompts,
  bearer tokens, API keys, story text, and image bytes are not exported.

For Langfuse, use its OTLP base endpoint (cloud or self-hosted) and provide the
documented Basic authorization plus ingestion-version header through
`OTEL_EXPORTER_OTLP_HEADERS`. The same setup works with an OpenTelemetry
Collector, Grafana Tempo, Jaeger, and other OTLP receivers. The health response
reports only `observability.otlp_traces` as `enabled` or `disabled`.

The dedicated [Observability and traces](observability.md) guide provides
copy-ready generic and Langfuse examples, sampling controls, verification, and
the data privacy boundary.

## Game and storage

- `interface.locale` controls terminal presentation only. Supported saved values
  are `en` and `it`; omit it or use `oneday config locale auto` to follow the OS.
  Browser interface language is stored separately in browser preferences.
- Interface locale never changes canonical story language or per-story TTS and
  voice-assignment language tags.
- `data_dir` contains `oneday.db`, generated visual/audio assets, and story data.
- `game.autosave_every` controls automatic save frequency.
- `game.typewriter_effect` and `typewriter_speed` affect terminal presentation.
- `game.visible_private_thoughts` is a storyteller/debug option and is off by default.
- `game.reward_budget` accepts `generous`, `balanced`, or `harsh`.

SQLite is canonical for both clients. Back up the entire data directory while
OneDay is stopped, or use a SQLite-safe backup procedure. The checked-in
`scripts/verify-sqlite-backup-restore.sh` utility creates a checksummed database
backup and restores only into an existing empty target after integrity and
foreign-key checks pass. Keep generated assets with that database backup.

For a standalone desktop profile, stop its local gateway from the Desktop
settings window (or quit the desktop application) before copying its profile's
`data/` directory. For a live terminal/server database, use a SQLite online
backup procedure instead of copying a changing database and WAL files. Restore
into an empty or stopped target data directory, preserve the directory as a
unit, then run the upgraded target and `oneday doctor` before serving it. Never
point a migration retry at the original after a failure: keep it stopped and
unchanged, restore into a separate empty recovery target, and promote only the
verified target. A backup restores its own profile only; it does not sync or
merge with another profile or remote server.
