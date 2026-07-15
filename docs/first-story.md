# Your first story

This guide takes OneDay from a fresh checkout to a persistent story. Choose the
terminal path for local Codex or Claude credentials; choose Docker for the full
browser interface with LiteLLM or OpenRouter.

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

Docker does not inherit host Codex or Claude logins. Configure LiteLLM or
OpenRouter instead:

```bash
cp config.example.yaml config.yaml
cp .env.example .env
```

In `config.yaml`, enable the provider and use its environment placeholder. Put
the real key in `.env`, then start OneDay:

```bash
docker compose pull
docker compose up -d
curl -fsS http://localhost:8788/api/health
```

Open `http://localhost:8788`. If the provider runs on the Docker host, replace
`127.0.0.1` with `host.docker.internal` in `config.yaml`.

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
