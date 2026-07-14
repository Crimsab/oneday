# Project roadmap

OneDay is actively developed. This roadmap describes improvement themes rather
than promised dates or a fixed release schedule. Released behavior is documented
in the main guides and [changelog](../CHANGELOG.md).

## Reliability

- Expand migration and restore fixtures across older public releases.
- Keep turn commits atomic across narrative, mechanics, media, and branch state.
- Improve provider capability detection, fallback diagnostics, and repair telemetry.
- Add longer-running story simulations for memory and canon regression testing.

## Player experience

- Continue keyboard, screen-reader, reduced-motion, and responsive-layout work.
- Improve branch navigation and comparison without obscuring the current story.
- Make complex world systems easier to inspect without turning the UI into an
  administration dashboard.
- Refine onboarding for local and hosted model providers.

## Portability and distribution

- Add reproducible packages for more platforms and architectures.
- Publish versioned container images after the source release workflow is stable.
- Improve import, export, backup, and recovery documentation and tooling.
- Keep the default Docker setup independent from any private infrastructure.

## Extension ecosystem

- Stabilize story-pack and minigame contracts with versioned examples.
- Document compatibility guarantees for third-party extensions.
- Grow reusable fixtures for custom genres, mechanics, and localization.

## Generated media

- Improve visual consistency across scenes, characters, and map locations.
- Preserve text-first gameplay when image, ASCII, map, or audio providers fail.
- Keep transparent map-symbol generation independent from general scene artwork.

Feature requests and focused proposals are welcome through
[GitHub Issues](https://github.com/Crimsab/oneday/issues). Before starting a
large change, open a discussion so its engine, persistence, and UI boundaries
can be agreed first.
