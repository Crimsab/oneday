# Your first story

This guide takes OneDay from a fresh checkout to a persistent story. Choose the
terminal path to use local Codex or Claude CLI credentials for narrative
generation. Choose Docker for the full browser interface. Its standard gateway
uses LiteLLM or OpenRouter for narrative generation, while an optional private
profile can reuse Codex OAuth for generated images.

## Terminal with Codex OAuth

Install Go 1.25.12 or use a release binary, then authenticate the Codex CLI:

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
codex login
go run ./cmd/oneday setup
go run ./cmd/oneday doctor
go run ./cmd/oneday
```

Choose **Codex OAuth** in setup. This route uses the local Codex login and does
not require `OPENAI_API_KEY`. RAG is optional; choose the Codex + local
embeddings setup if Ollama is available.

## Browser with Docker

The standard gateway image does not contain the Codex or Claude CLI. Use a
LiteLLM-compatible endpoint or OpenRouter for narrative generation. If you also
want Codex OAuth image generation, follow the
[optional imagegen-bridge profile](docker.md#optional-codex-oauth-imagegen-bridge-profile).
Its helper copies only the host Codex `auth.json` into a dedicated volume; it
does not mount your home directory or expose the OAuth file to the OneDay
gateway.

```bash
git clone https://github.com/Crimsab/oneday.git
cd oneday
docker compose pull
docker compose run --rm oneday-tools docker init
docker compose up -d
docker compose run --rm oneday-tools docker token
```

Open `http://localhost:8788` and enter the generated browser credential. Open
**Setup**, configure one narrative provider, and run its readiness check.

```bash
curl -fsS http://localhost:8788/api/health
```

If no published container tag is available, use the current-source commands in
[Getting started](getting-started.md#browser-with-docker). If the provider runs
on the Docker host, use `host.docker.internal` instead of `127.0.0.1` in its
URL.

## Create the world

From the story wizard:

1. Describe any premise: genre, setting, protagonist, desired themes, and
   anything the story must avoid.
2. Choose the story language, tone, difficulty, and visual direction. The story
   language is independent from the application interface language.
3. Keep combat off when it does not fit. Challenges, investigations, crafting,
   social encounters, and minigames work without a combat-focused story.
4. Review the generated foundation, then create the story.
5. Pick a suggested action or write any action in your own words.

The turn is committed atomically to SQLite. Characters, relationships,
inventory, clues, projects, factions, locations, and consequences survive
restarts and stay scoped to the active branch.

## Optional media

Images and speech are disabled in the public template until a provider is
configured. They are never required to play: canonical text, map topology, and
accessibility labels remain available when media generation is disabled or
fails. See [Generated media](media.md) when you are ready to enable them.

## Next steps

- Explore the complete [feature tour](features.md).
- Learn how [story systems](story-systems.md) affect turns.
- Configure providers, RAG, media, and traces in [Configuration](configuration.md).
- Use [Troubleshooting](troubleshooting.md) if provider or gateway checks fail.
