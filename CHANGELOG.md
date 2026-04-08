# Changelog

## [1.2.0](https://github.com/Crimsab/oneday/compare/v1.1.0...v1.2.0) (2026-04-08)


### Features

* **tui:** complete phase 9 narrative ux polish ([213bfad](https://github.com/Crimsab/oneday/commit/213bfad3776441a851f17d565c4b42dbe5660e66))

## [1.1.0](https://github.com/Crimsab/oneday/compare/v1.0.1...v1.1.0) (2026-04-08)


### Features

* **story:** add guided new story wizard ([caf588d](https://github.com/Crimsab/oneday/commit/caf588da8457851a2c117c8963ac4822f7eb1223))
* **tui:** add streamed narrative telemetry ([3083d6a](https://github.com/Crimsab/oneday/commit/3083d6a58778628dd0464cb2ce2583994689bd88))

## [1.0.1](https://github.com/Crimsab/oneday/compare/v1.0.0...v1.0.1) (2026-04-08)


### Bug Fixes

* **tui:** avoid builder copy panics and improve config lookup ([d8541c5](https://github.com/Crimsab/oneday/commit/d8541c5b2556040838879bf1033483c729d580aa))

## 1.0.0 (2026-04-08)


### Features

* **achievements+mood:** implement Phase 7-01 achievement engine and mood theming ([9cfa839](https://github.com/Crimsab/oneday/commit/9cfa8394f2e0f7f6d72217ca55fbd68ac8f276fe))
* **achievements:** phase 7-02 integration and polish pass ([859043d](https://github.com/Crimsab/oneday/commit/859043d53c8941fcf567220532c04c12ac893e36))
* **ai:** add structured outputs and oneday benchmarks ([5970e9f](https://github.com/Crimsab/oneday/commit/5970e9f214e014863d4fcab5f8f0347347be5370))
* **ai:** implement AI provider router with fallback chain ([fe2681f](https://github.com/Crimsab/oneday/commit/fe2681ffa6de273e6d3ee8a07b21f2a44107a04e))
* **ai:** streaming support, response parser, and typewriter component ([11ecebd](https://github.com/Crimsab/oneday/commit/11ecebd9df1191934fc0abf1f48010c5d2b066a5))
* **challenge-tui:** add challenge overlays and narrative integration ([14d299e](https://github.com/Crimsab/oneday/commit/14d299eec964b2cc09d23fd223b00f9eef2cd0ad))
* **commands:** implement slash command system (/help, /inventory, /stats, /save, /load, /quit) ([6a1a510](https://github.com/Crimsab/oneday/commit/6a1a5104bf0cdf2281574527cc316e4e32a20566))
* **config:** add configuration system with YAML parsing and provider priority chain ([5841791](https://github.com/Crimsab/oneday/commit/5841791ef105171ecffa1420284d284a2fd535b5))
* **docs:** add chat commands system and /narrator command ([7954bd6](https://github.com/Crimsab/oneday/commit/7954bd6c97a969ce7b2be3fddda22e9a1bd2332f))
* **engine:** /map, /journal, /achievements commands + dynamic world updates + context enrichment ([91b92bc](https://github.com/Crimsab/oneday/commit/91b92bc026bff1727ffe8508a9da48d558aa95d7))
* **engine:** challenge engine tests + combat engine + mini-games (plan 6.1) ([8e5e0f0](https://github.com/Crimsab/oneday/commit/8e5e0f0e90426021a1b5944b12e96a1219190fb5))
* **engine:** chapter system, sub-sessions, and /narrator command ([de25dd1](https://github.com/Crimsab/oneday/commit/de25dd1e329a901e7a70be29b1d454a32ebee3c9))
* **engine:** character growth engine and NPC generation/persistence (plan 4.1) ([2cda4f1](https://github.com/Crimsab/oneday/commit/2cda4f1678a7f74474d69a5f2179aab3df7f7356))
* **engine:** full game loop, state changes, save/load system, load story menu ([5a5cc3d](https://github.com/Crimsab/oneday/commit/5a5cc3d7a82e5678239c03a01286f4c6f7d4f36a))
* **engine:** session management, JSONL persistence, and context builder ([7122712](https://github.com/Crimsab/oneday/commit/71227120717c9658d9361e1d53e404d7d5f9c89a))
* initial project setup ([4e69865](https://github.com/Crimsab/oneday/commit/4e69865e3178e94a68f58bd8f047b32b2bdcc391))
* **rag:** implement embedding client, vector store, and periodic summarization ([8043cf2](https://github.com/Crimsab/oneday/commit/8043cf238c300eb521d28d79380e02493fd0c9f6))
* **storage:** add SQLite storage layer with migrations and models ([1dd2768](https://github.com/Crimsab/oneday/commit/1dd27687e57ec409c876873631c8ed00c45901ea))
* **story:** add authoring controls and release automation ([4da784b](https://github.com/Crimsab/oneday/commit/4da784b35ff0240c053a6ff27dd431bccd57bd72))
* **story:** implement AI-guided story creation flow ([dcdd9dd](https://github.com/Crimsab/oneday/commit/dcdd9dd814fce9bbe1532d3520f3d6a015e91964))
* **tui:** add Bubbletea app skeleton, main menu, and theme system ([a1a2ee9](https://github.com/Crimsab/oneday/commit/a1a2ee9b2c9a36eb36b737cbfa33c998efbddaa5))
* **tui:** combat TUI view, crafting engine, and crafting TUI view ([d29ad39](https://github.com/Crimsab/oneday/commit/d29ad39d85a15e1ee8e53f73c884fec629083f2c))
* **tui:** complete phase 8 rendering polish ([b300298](https://github.com/Crimsab/oneday/commit/b30029821217a8edc2a5d03edd17899ec07cab49))
* **tui:** enhanced /stats and /inventory overlays with rich display ([8cb8a56](https://github.com/Crimsab/oneday/commit/8cb8a5628cf431acb8a9ffa460dc3d3cd6f12986))
* **tui:** implement narrative view with typewriter, choices, status bar, and narrator engine (plan 2.4) ([a7368d9](https://github.com/Crimsab/oneday/commit/a7368d9214909982f9d67c67cff5f7f420326d88))


### Bug Fixes

* **app:** restore local ai config and settings screen ([bff129c](https://github.com/Crimsab/oneday/commit/bff129c3c864fc8fe53e308e11e9dfc7232744da))
* **ci:** unblock release-please and avoid artifact quota failures ([c4fa0c8](https://github.com/Crimsab/oneday/commit/c4fa0c8151f57296e73894b97ed3d1bd85603ae4))
* **engine:** bugfix and stabilization phase 6.1 ([c568dd4](https://github.com/Crimsab/oneday/commit/c568dd4ca628c7379a8127c8505dbcf315b1802b))
* **storage:** update idempotent test for multi-migration schema ([c70214e](https://github.com/Crimsab/oneday/commit/c70214edf81baacb4f14742ffe3fd62da6b691c8))
* **tui:** add markdown rendering + hardcoded welcome message ([f738ee9](https://github.com/Crimsab/oneday/commit/f738ee9657ce166458724e99012b63f1facafac8))
* **tui:** fix strings.Builder copy panic in newstory view ([a330780](https://github.com/Crimsab/oneday/commit/a330780a0e3219dfec991d0d340f4bda7e44362d))
