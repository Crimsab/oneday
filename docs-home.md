---
hide:
  - toc
---

# OneDay documentation

<figure class="oneday-hero">
  <img
    src="docs/assets/oneday-hero.webp"
    alt="OneDay turns a premise into a persistent world with branches, relationships, generated media, crafting, and minigames."
  >
</figure>

<div class="oneday-lead" markdown>

Create persistent AI stories that remember choices, relationships, inventory,
and alternate branches. Play in a browser, desktop window, or terminal.

[Start with Docker](docs/getting-started.md#browser-with-docker){ .md-button .md-button--primary }
[Create your first story](docs/first-story.md){ .md-button }

</div>

## Choose how to run OneDay

<div class="grid cards oneday-paths" markdown>

-   **Browser or PWA**

    ---

    Use Docker on Windows, macOS, or Linux. Docker builds the complete app and
    keeps the story database in a named volume.

    [Open the Docker quick start](docs/getting-started.md#browser-with-docker)

-   **Desktop**

    ---

    Connect a native window to an existing HTTPS server, or use an available
    standalone package for a separate local profile.

    [Understand desktop profiles](docs/desktop.md)

-   **Terminal**

    ---

    Run the Go client from source or use a release binary. This path supports
    local Codex and Claude CLI sessions.

    [Open the terminal quick start](docs/getting-started.md#terminal-client)

</div>

!!! info "What you need"

    OneDay needs one narrative AI provider. Generated images and speech are
    optional. You can add them after the first text turn works.

## First successful run

1. [Install OneDay](docs/getting-started.md).
2. Configure one narrative provider in **Setup** or with `oneday setup`.
3. Run the readiness checks.
4. [Create the first story](docs/first-story.md).
5. Back up the canonical data store before you move or update it.

Stories do not synchronize automatically between Docker, terminal, remote
desktop, and standalone desktop profiles. Decide where a story will live before
you create it.

## Find the right guide

| Goal | Guide |
| --- | --- |
| Configure providers, models, authentication, or storage | [Configuration](docs/configuration.md) |
| Update, back up, or restore Docker | [Docker deployment](docs/docker.md) |
| Configure images, maps, or speech | [Generated media](docs/media.md) |
| Understand branches, challenges, crafting, or investigations | [Story systems](docs/story-systems.md) |
| Diagnose startup, login, or provider failures | [Troubleshooting](docs/troubleshooting.md) |
| Build or contribute to OneDay | [Development](docs/development.md) |

For the complete index, open the [documentation map](docs/README.md).
