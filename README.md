# OneDay

An AI-driven text RPG played entirely in the terminal.

Stories are infinite, AI-generated, and deeply personalized. Every NPC has personality, desires, and opinions about you. Every choice matters. Every story is unique.

## Features

- **AI-Powered Narrative** — stories are generated and continued by AI models (Claude, GPT, Gemini, etc.) via configurable provider chain
- **Dynamic Everything** — stats, skills, NPCs, items, locations, objectives, achievements are all generated at runtime. Nothing is hardcoded
- **Deep Character System** — start from nothing, earn everything. Traits emerge from your actions, skills from your practice, titles from your deeds
- **Living NPCs** — each NPC has personality, speech patterns, quirks, private thoughts about you, and hidden desires that drive their behavior
- **Always Free Action** — you can always type your own action, not just pick from a list. Talk to the boss instead of fighting. Craft something absurd. Try anything
- **Turn-Based Combat** — narrative-driven with stat checks, dice rolls, and creative solutions. Chat is always present, even mid-fight
- **AI Crafting** — no hardcoded recipes. Describe what you want to build, the AI evaluates if it makes sense with your materials and skills
- **Challenge System** — dice rolls, stat checks, skill checks, rock-paper-scissors, and other mini-games. The engine rolls, not the AI
- **AI Achievements** — no predefined list. The AI recognizes noteworthy moments and awards unique achievements
- **Persistent Memory** — full chat history saved, RAG-powered long-term context. Stories can be infinite
- **Multiple Genres** — fantasy, cyberpunk, horror, slice-of-life, anything. Each genre defines its own stat system and rules

## Tech Stack

- **Go** + Bubbletea/Bubbles/Lipgloss (TUI)
- **SQLite** + sqlite-vec (storage + vector search)
- AI via **Claude Code** / **LiteLLM** / **OpenRouter** (configurable fallback chain)

## Quick Start

```bash
# Build
go build -o oneday ./cmd/oneday

# Run
./oneday

# Windows
GOOS=windows GOARCH=amd64 go build -o oneday.exe ./cmd/oneday
```

## Configuration

Copy `config.example.yaml` to `config.yaml` and set your AI provider endpoints/keys.

## License

Personal project.
