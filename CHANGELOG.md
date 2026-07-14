# Changelog

## [1.7.0](https://github.com/Crimsab/oneday/compare/v1.6.0...v1.7.0) (2026-07-05)


### Features

* **ai:** support litellm responses streaming ([51cd0c2](https://github.com/Crimsab/oneday/commit/51cd0c29d6c152ec57c99ffea5212a86c1c44606))
* **browser:** add collapsible sidebars ([14d0459](https://github.com/Crimsab/oneday/commit/14d04591e1c733827c634b9532da0b194e343f5a))
* **browser:** add command palette completions ([45cd3f8](https://github.com/Crimsab/oneday/commit/45cd3f8dbe8d8df77bb3d296524eb3d12ac2d310))
* **browser:** add markdown path chapters ([a2a3372](https://github.com/Crimsab/oneday/commit/a2a3372eb56db1d58e0c6c4dd24f3067868810b6))
* **browser:** add Rust realtime gateway ([ad07e86](https://github.com/Crimsab/oneday/commit/ad07e86ad602e4a2f37cbc9946dc44166e868291))
* **browser:** add save deletion and load filtering ([a8324c3](https://github.com/Crimsab/oneday/commit/a8324c3d8ee2522d6c5896d0a41591f06841c650))
* **browser:** add shared turn bridge and lock ([671b0a4](https://github.com/Crimsab/oneday/commit/671b0a4ea02157f5233631654f87d13343ab4c2a))
* **browser:** add story creation and decision stack ([63d6568](https://github.com/Crimsab/oneday/commit/63d65683b2824e591df04a8ce587057418c07097))
* **browser:** add story CRUD controls ([db81091](https://github.com/Crimsab/oneday/commit/db810911fdeef4be92754c9ed403fbef6ae25710))
* **browser:** expand inspector modules ([d91b944](https://github.com/Crimsab/oneday/commit/d91b944bcd7405cadfdf9907aed437524fa72295))
* **browser:** expose meta and save gateway ops ([610e61c](https://github.com/Crimsab/oneday/commit/610e61ccd1daebb898514db3d774e283911cc4c1))
* **browser:** migrate gateway UI to React cockpit ([087cb51](https://github.com/Crimsab/oneday/commit/087cb518a3393fe50a0beb8136a848f09ccc2f41))
* **browser:** preview story delete impact ([8303ae4](https://github.com/Crimsab/oneday/commit/8303ae4ef587f81f0c1d5a08f59a59451c79dcd6))
* **browser:** save shared model routing ([ed60a2c](https://github.com/Crimsab/oneday/commit/ed60a2ca4ee88e7b0462c33d8c011b09a2b8cc91))
* **browser:** share command descriptors ([63e1b22](https://github.com/Crimsab/oneday/commit/63e1b22a0092ce4a154e51f3e6a6ea421b7c35f6))
* **browser:** stream turn lifecycle events ([422e80e](https://github.com/Crimsab/oneday/commit/422e80eb0d7921bf46dbb00fd3b339ddb7f5162a))
* **browser:** surface model routing settings ([b0e4afa](https://github.com/Crimsab/oneday/commit/b0e4afa1fedd32f50e649b805c69201c385cb562))
* **browser:** sync terminal with gateway turns ([d9978fc](https://github.com/Crimsab/oneday/commit/d9978fc1f378316caa3eabb135b3e968ec072403))
* **browser:** wire modules preferences and history ([4f822da](https://github.com/Crimsab/oneday/commit/4f822da8f2915cd7dc24f7afc6ec5cec150aa9dd))
* **config:** centralize model provider settings ([b77caba](https://github.com/Crimsab/oneday/commit/b77cabad5e5898e64bb186491c8ddf2be9fb7435))
* **config:** expose shared image generation settings ([95786b7](https://github.com/Crimsab/oneday/commit/95786b75be2c9f45c227083be6f77e730b23af7f))
* **engine:** track progressive npc discovery ([ca67aad](https://github.com/Crimsab/oneday/commit/ca67aad3fff820f44dc520f2bf54cfca20b71fa9))
* **game:** checkpoint gameplay audit fixes ([860fc20](https://github.com/Crimsab/oneday/commit/860fc20d1ce3455c7779726f3ebdd1f62c165a7d))
* **gateway:** add browser meta and save routes ([648d677](https://github.com/Crimsab/oneday/commit/648d677b8791276dd229d5eda2c07431ebebe2c2))
* **gateway:** add visual asset profile contract ([8aefc3f](https://github.com/Crimsab/oneday/commit/8aefc3fc19dcf178c9bfe2786a88265dfbe63552))
* **gateway:** gate npc portraits by discovery state ([0c7929b](https://github.com/Crimsab/oneday/commit/0c7929b8c77668bc7f839e90c12fd7dd66e5ed3d))
* **gateway:** stream browser turn events live ([7cfc940](https://github.com/Crimsab/oneday/commit/7cfc9402880d464b5572d8c759497569aa85f892))
* **gateway:** stream visual generation updates ([d04050f](https://github.com/Crimsab/oneday/commit/d04050ff9368e78c7bbaa52e7be2bb125ac3b58b))
* **imagegen:** expose visual generation jobs ([7b237fc](https://github.com/Crimsab/oneday/commit/7b237fce0ac69c114c86948af3a1ab344624ad57))
* **imagegen:** harden visual job lifecycle ([ce9fb79](https://github.com/Crimsab/oneday/commit/ce9fb7941be60924be08572ba5031d0448a906da))
* **imagegen:** queue visual generation jobs ([1702c72](https://github.com/Crimsab/oneday/commit/1702c72c6316630f4745abdd78af6c6dcdcbba58))
* **web:** add terminal-compatible story wizard ([bc69d0e](https://github.com/Crimsab/oneday/commit/bc69d0efa662ee374bb77474f9c48665c23071a8))
* **web:** add visual asset editor ([557b367](https://github.com/Crimsab/oneday/commit/557b36776dd5c2afdb1eee5aa83ac8f58b4924c3))
* **web:** generate visual assets on demand ([152ba21](https://github.com/Crimsab/oneday/commit/152ba2129f5e808c8a15d35366a8419c3a4916f4))
* **web:** surface npc discovery state ([84cf691](https://github.com/Crimsab/oneday/commit/84cf6916b91b73f2f25aca7a84b18a1b92bc7400))
* **web:** wire meta commands and save management ([77c44f8](https://github.com/Crimsab/oneday/commit/77c44f838c4de846b29e654f8c0897f189fba0d6))
* **web:** wire visual generation through OpenClaw ([f64a1e2](https://github.com/Crimsab/oneday/commit/f64a1e23ee27ff12609280b284c5ce2e3bdc2c93))


### Bug Fixes

* **ai:** require profiles for newly named npcs ([5361fd3](https://github.com/Crimsab/oneday/commit/5361fd3214615aa3b795df7fbda1fc1484404359))
* **browser:** add craft module and compact choices ([3c29ec9](https://github.com/Crimsab/oneday/commit/3c29ec9a17ac9544198353a1aa2c9bc1977c9a36))
* **browser:** apply oracle phase zero hardening ([a612f07](https://github.com/Crimsab/oneday/commit/a612f072aa860866410e62bc9d7286afb0a9f726))
* **browser:** clean inspector empty state rendering ([1f6fcd0](https://github.com/Crimsab/oneday/commit/1f6fcd0138500da9d912341feb0b83f4b91ff0e8))
* **browser:** clean save timestamps in UI ([2f7606a](https://github.com/Crimsab/oneday/commit/2f7606aed6e4a63d2eabbdcdc21ee4641b64ae31))
* **browser:** compact choice effect copy ([26c9a90](https://github.com/Crimsab/oneday/commit/26c9a90ad4f7c77b651d15f7067450537a2f2154))
* **browser:** contain mobile gateway overflow ([639d9f7](https://github.com/Crimsab/oneday/commit/639d9f7a69bf1d4d49493afa4a881e5013641662))
* **browser:** copy gateway binary from release build ([a8b013b](https://github.com/Crimsab/oneday/commit/a8b013b6b799d13a189823762285d9a888100d2a))
* **browser:** guard shared turn and config races ([6128d77](https://github.com/Crimsab/oneday/commit/6128d772f4ca7f84b3bdb6631565c654f4ef913a))
* **browser:** harden choice submission and metadata cards ([d40f18b](https://github.com/Crimsab/oneday/commit/d40f18b414225903385e74987b441411e9391250))
* **browser:** harden command parity and JSON display ([fbf669b](https://github.com/Crimsab/oneday/commit/fbf669b85b81083708b5b969bda2d5d82ffa14a2))
* **browser:** harden player state and choice UX ([e883929](https://github.com/Crimsab/oneday/commit/e883929b5ee202a3d8b11ddb3ba56add848ea4f4))
* **browser:** harden shared model routing ([264faff](https://github.com/Crimsab/oneday/commit/264faff350df402d66031fc25577b2dc28ce2d35))
* **browser:** improve command completion ranking ([ca60b46](https://github.com/Crimsab/oneday/commit/ca60b46690dbec72eacef13a29d98bef63f254a2))
* **browser:** improve gateway responsive layout ([e2029d9](https://github.com/Crimsab/oneday/commit/e2029d90856943daf29087b8653a24796d9c7032))
* **browser:** merge backend command descriptors with local aliases ([1d006f8](https://github.com/Crimsab/oneday/commit/1d006f8c12a199359feba74fe8876d473f53968c))
* **browser:** place choices inline and close command palette ([ff2d40f](https://github.com/Crimsab/oneday/commit/ff2d40f197b232b60f167326fec234c044f782ea))
* **browser:** polish scrollbars and inspector text ([eb21ef2](https://github.com/Crimsab/oneday/commit/eb21ef25ddbc5515e9c1186f84c6ce8de3c81340))
* **browser:** remember local commands and clean timestamps ([582860b](https://github.com/Crimsab/oneday/commit/582860b2003fa24d120d97d8fc595f11b1267475))
* **browser:** tighten choice effect labels ([175b726](https://github.com/Crimsab/oneday/commit/175b726497cce624b17694843ee8ff2ceebdea63))
* **browser:** tighten cockpit layout and slash commands ([5a54800](https://github.com/Crimsab/oneday/commit/5a54800edc77cc0674be824d4ad6ee6a095c8e3a))
* **browser:** tighten mobile panel controls ([56c5c08](https://github.com/Crimsab/oneday/commit/56c5c082b28620d34a3e5734cfa0829dcdee9378))
* **browser:** use compatible gateway runtime image ([5b5ba73](https://github.com/Crimsab/oneday/commit/5b5ba7327c5563cabc7ab7677b5d925988a2f239))
* **ci:** split checks and support windows config builds ([5ff2bfd](https://github.com/Crimsab/oneday/commit/5ff2bfd42d82f7d39e6736f59b91210342f698c1))
* **ci:** stabilize workflow lint and web contract tests ([3bb389a](https://github.com/Crimsab/oneday/commit/3bb389aafb6ef6743ef1daee7372604b3cb26899))
* **config:** align imagegen compose defaults ([a327bfc](https://github.com/Crimsab/oneday/commit/a327bfc4a72b687a574737ef29bb3c2c52788c9c))
* **docker:** let config drive imagegen defaults ([011fcee](https://github.com/Crimsab/oneday/commit/011fcee39db25ffed65e7c8c7363ce21a2401b3a))
* **engine:** add local anti-loop progression fallback ([9c5fed0](https://github.com/Crimsab/oneday/commit/9c5fed0d95a3720008f10e3b48f23530613b57cc))
* **engine:** canonicalize unknown npc updates ([1e66e12](https://github.com/Crimsab/oneday/commit/1e66e12e1fb3efac9bd79b385373ec48c10f3afd))
* **engine:** defer narrator-managed state keys ([1fea0df](https://github.com/Crimsab/oneday/commit/1fea0dfe658b480531e0a5b97ab21f218d4a1e7d))
* **engine:** guard shared story mutations by revision ([64e5fa9](https://github.com/Crimsab/oneday/commit/64e5fa977fbdd3c24dd7ba405dcf77ae420e2053))
* **engine:** infer npcs from narrative mentions ([a236e33](https://github.com/Crimsab/oneday/commit/a236e3379caf6c539e162f61b662452553883406))
* **engine:** make story mutations transactional ([5636d38](https://github.com/Crimsab/oneday/commit/5636d389f53e116ef37f8e9e7c8ade94eee7a640))
* **engine:** preserve story context for empty sessions ([d4187ca](https://github.com/Crimsab/oneday/commit/d4187caf96b9324b8faffe8eed68aaab0c521143))
* **engine:** renew story mutation locks ([eef3bba](https://github.com/Crimsab/oneday/commit/eef3bba81e1ecea19689f4f8db7981f35852e946))
* **engine:** track investigation suspects as npcs ([990c26d](https://github.com/Crimsab/oneday/commit/990c26da1603e1b3387bb2a64fef67717bab2ff3))
* **gateway:** include public visual assets in web build ([ddd26bc](https://github.com/Crimsab/oneday/commit/ddd26bca89c2200eb655a43717fb821744f5044c))
* **gateway:** make SSE snapshot polling retry-safe ([1e7e0aa](https://github.com/Crimsab/oneday/commit/1e7e0aa21cfdafc77dab63b1bf5d81e73f192edf))
* **gateway:** mount codex cli for browser turns ([8b9bd99](https://github.com/Crimsab/oneday/commit/8b9bd998ed52cb50669351b0af482564aa14cef0))
* **gateway:** surface next-turn choices ([c6a05eb](https://github.com/Crimsab/oneday/commit/c6a05eb3f5b650cf2b2873ddaa1b3fa055436b96))
* **imagegen:** cancel visual jobs on story deletion ([508db8f](https://github.com/Crimsab/oneday/commit/508db8fdcc6faf188c7d5e951449fb36fa9f2757))
* **imagegen:** keep prior asset ready on regen failure ([94eacda](https://github.com/Crimsab/oneday/commit/94eacda26434e1e4089e13f09ebaca174c945067))
* **imagegen:** normalize openclaw image model ([c17f615](https://github.com/Crimsab/oneday/commit/c17f6158a7fbd55898aae97b5349575595484f5e))
* **parity:** harden streams and choice actions ([c2debdf](https://github.com/Crimsab/oneday/commit/c2debdf1f8c59086f3b0dd772846473642097744))
* **streaming:** harden responses and gateway events ([0d63e59](https://github.com/Crimsab/oneday/commit/0d63e59d89f19d5c3fb887c89c704763a765db7f))
* **sync:** broaden snapshot version and choice freshness ([b9988af](https://github.com/Crimsab/oneday/commit/b9988afb26ad820a19a67aa48e3dae2b45d6f1da))
* **sync:** keep visual asset refresh idempotent ([8447b88](https://github.com/Crimsab/oneday/commit/8447b88aa8b3ba5d157b5d99b254727ff8a0d866))
* **web:** balance desktop choice grid ([a2b0ee3](https://github.com/Crimsab/oneday/commit/a2b0ee3214516346e664ddfb1113e447a895b193))
* **web:** harden browser action safety ([c325222](https://github.com/Crimsab/oneday/commit/c3252224f337b3e8afc9e3ba77c47256355ac774))
* **web:** isolate codex modal tab from inspector ([447230c](https://github.com/Crimsab/oneday/commit/447230c420c3662c55cf7fd685d6a0130d0d442d))
* **web:** keep active story reloadable ([c0f2f8b](https://github.com/Crimsab/oneday/commit/c0f2f8bd4a23debe86ed9a45e686dab9999dd697))
* **web:** keep choice cards in a stable grid ([ac65b3e](https://github.com/Crimsab/oneday/commit/ac65b3e66085cd9012e6c62827e04a457893fda8))
* **web:** submit composer textarea value ([0562727](https://github.com/Crimsab/oneday/commit/05627276200c3e9d290db895ecefb493fe7f5de6))
* **web:** suppress raw structured stream previews ([c464f92](https://github.com/Crimsab/oneday/commit/c464f9292968476e5c87a341f6027ed27e41f8cc))
* **web:** tighten composer and choice layout ([b6ff04b](https://github.com/Crimsab/oneday/commit/b6ff04bfbabface7048792a23f18708401fd3b26))

## [1.6.0](https://github.com/Crimsab/oneday/compare/v1.5.0...v1.6.0) (2026-05-13)


### Features

* **ai:** add reconfigurable local rag setup tooling ([32b5a3e](https://github.com/Crimsab/oneday/commit/32b5a3ecc737073dcf35c74ab6d9b7a94907901f))
* **config:** add version migration defaults ([1574d7b](https://github.com/Crimsab/oneday/commit/1574d7b62f492598811281bc4dc19837109cea61))
* **doctor:** add safe json diagnostics ([f9f4dc3](https://github.com/Crimsab/oneday/commit/f9f4dc3f505d55438d5de695310dec8ca33bbce6))
* **export:** add friend-safe package command ([1141c4f](https://github.com/Crimsab/oneday/commit/1141c4f80f81650f6f28cd14363a6a9fda2cc5b9))
* **narrative:** harden runtime continuity and autocomplete flows ([3f5166d](https://github.com/Crimsab/oneday/commit/3f5166d90f0307edfd7c35cccf97a5c1eb9d3ff1))
* **rag:** add benchmark guidance ([24de437](https://github.com/Crimsab/oneday/commit/24de4373e7853c4bbb1def70ecfde90e6ed7063f))
* **rag:** add safe reindex maintenance command ([3ea8e93](https://github.com/Crimsab/oneday/commit/3ea8e936dd9ba049629051cde7c9a1e077f95764))
* **story-packs:** validate and select pack scaffolds ([596b8e8](https://github.com/Crimsab/oneday/commit/596b8e869e23f23e351df58aa1e47e720ce5c830))

## [1.5.0](https://github.com/Crimsab/oneday/compare/v1.4.0...v1.5.0) (2026-04-14)


### Features

* **narrative:** improve pacing, timeline, and multiline input ([b57507f](https://github.com/Crimsab/oneday/commit/b57507f377502126d27e12880cd848ec441f0ced))

## [1.4.0](https://github.com/Crimsab/oneday/compare/v1.3.0...v1.4.0) (2026-04-09)


### Features

* **ai:** harden structured json repair and runtime ux ([42fdf35](https://github.com/Crimsab/oneday/commit/42fdf3545f9d2fd2ab605427bc8421a3c950ba27))
* **ai:** tune repair fallback strategy ([2d0f2f4](https://github.com/Crimsab/oneday/commit/2d0f2f4e89dc6bc37f2665814f1937204a3d3ab5))
* **build:** stamp runtime version provenance ([04fb6c7](https://github.com/Crimsab/oneday/commit/04fb6c73f72a629542385e66f060140a634455f7))
* **codex:** integrate investigation board browsing ([58217ba](https://github.com/Crimsab/oneday/commit/58217baa8c38905ca0d4ea0ff3e83cde964232b4))
* **codex:** surface fronts in dossiers ([6364506](https://github.com/Crimsab/oneday/commit/63645060fde9d1b8f382a7aae3ccfbf01601d3a5))
* **codex:** trace nemesis escalation without leaks ([8887879](https://github.com/Crimsab/oneday/commit/8887879743b0ad916a7a56255825772f85d7950c))
* **fronts:** add canonical front state ([73e98cf](https://github.com/Crimsab/oneday/commit/73e98cfd3b65faaaa33bda4ba6761e29a72c0035))
* **fronts:** add dedicated fallout tracker ([6d1c7ac](https://github.com/Crimsab/oneday/commit/6d1c7ac52772c4dfec413c49e9bf3e8c7f74a0f4))
* **fronts:** derive regional pressure fallout ([ed618dc](https://github.com/Crimsab/oneday/commit/ed618dcf0e3de269f480bc63379623bcc7c9bbff))
* **fronts:** sync continuity with fail-forward ([cc8cc3d](https://github.com/Crimsab/oneday/commit/cc8cc3d91c236c5f3938a984bf33151791c3f53a))
* **gameplay:** add chapter guidance and stronger combat triggers ([1c47a72](https://github.com/Crimsab/oneday/commit/1c47a72b382bac5e93260a811ea84b1430394e2f))
* **investigation:** add canonical board persistence ([64e8476](https://github.com/Crimsab/oneday/commit/64e847688a30afe05b51460155a57230487fd43c))
* **investigation:** normalize evidence and theory updates ([8e184b2](https://github.com/Crimsab/oneday/commit/8e184b26927b0a6d7ca2b1519d45eb888c004708))
* **investigations:** add dedicated mystery workspace ([f28f33c](https://github.com/Crimsab/oneday/commit/f28f33cd74c45467419f5271f51378728ede0e13))
* **nemesis:** add canonical rival promotion state ([d04a4e1](https://github.com/Crimsab/oneday/commit/d04a4e151c77763cc67dd779ae32beea66f52c93))
* **nemesis:** add multi-outcome resolution fallout ([5aa4b86](https://github.com/Crimsab/oneday/commit/5aa4b86aea9b690cf41f77d3f2c988eb1d25e870))
* **nemesis:** prioritize recurring rivals in context ([41070f4](https://github.com/Crimsab/oneday/commit/41070f4e3fc660c2887023baea515a719637d6d6))
* **oneday:** add codex archive and runtime polish ([ff5d37d](https://github.com/Crimsab/oneday/commit/ff5d37dc9aee8ea64c839bb828cb0575f6bb078a))
* **projects:** add canonical downtime clock persistence ([2786af1](https://github.com/Crimsab/oneday/commit/2786af1017ee0db5f330bc896cef9505307827b9))
* **projects:** add dedicated project workspace ([85c8f80](https://github.com/Crimsab/oneday/commit/85c8f806e6cb9d836da04c2cf9cc996559f48061))
* **projects:** surface downtime outcomes in codex ([fbcc244](https://github.com/Crimsab/oneday/commit/fbcc24487bc6e26eea30ac154635a57ac2071c02))
* **projects:** tie downtime clocks to pressure and cost ([1ac45c0](https://github.com/Crimsab/oneday/commit/1ac45c0c77c25ce8964da9e65f8be9a3088bc23d))
* **social-duel:** add duel runtime and tui flow ([c56e2da](https://github.com/Crimsab/oneday/commit/c56e2da9b0806405cf48d2626fb3bacb62728588))
* **social-duel:** add engine state machine ([5e7b958](https://github.com/Crimsab/oneday/commit/5e7b95856d550d351e69e142574bde2659839703))
* **social-duel:** add narrator contract metadata ([77ae07a](https://github.com/Crimsab/oneday/commit/77ae07ad42371eef761eeafc15c160057d9b78b4))
* **social-duel:** persist aftermath into world state ([3eda9d5](https://github.com/Crimsab/oneday/commit/3eda9d5d4574928d14ad051f673e1a1900a438f4))
* **tui:** add talk one-shot and input history ([ae49e7c](https://github.com/Crimsab/oneday/commit/ae49e7cdfad6a159091aea77e15cc3348b8926e6))
* **tui:** complete phase 11 runtime reliability ([3bbb201](https://github.com/Crimsab/oneday/commit/3bbb2010a2452094034657846a4a87f2cb8b9716))
* **ux:** unify active system navigation ([2a99e8d](https://github.com/Crimsab/oneday/commit/2a99e8d1b4bd5d9cac5f4cd02a48f2471f3cc03d))
* **world:** execute phase 13 living world systems ([5874733](https://github.com/Crimsab/oneday/commit/587473379154bc2ecf890df50ab0f4805d66c6b8))


### Bug Fixes

* **ai:** align structured stream and sync requests ([f3e4056](https://github.com/Crimsab/oneday/commit/f3e4056e4a4428ab4ea21b48d81567e28f768de4))
* **ascii:** retry fallback model and surface failures ([4c71cea](https://github.com/Crimsab/oneday/commit/4c71cea18456876e744cd5ed4ab26127bd2bc975))
* **config:** make embedding provider selection explicit ([7bcf023](https://github.com/Crimsab/oneday/commit/7bcf0237a65faa9fe8ed26bf3a32d9b11fc585db))
* **gameplay:** improve persistence and history UX ([0c2db51](https://github.com/Crimsab/oneday/commit/0c2db5189a0bda4f0583ff0227aa5aa8181ade60))
* **phase-15:** harden canonical turn commits ([02ac7fb](https://github.com/Crimsab/oneday/commit/02ac7fbd0021a5e82b6c138c4c065d1f0eec2adc))
* **rag:** index summaries by committed turns ([b404651](https://github.com/Crimsab/oneday/commit/b404651098fb71cb85491b493efad95fdcf8f5e1))
* **rendering:** normalize dialogue prose extraction ([08f9a83](https://github.com/Crimsab/oneday/commit/08f9a83331c5d1cd633ebccdd9f31452a5c0293c))
* **state:** add phase 12 rollback artifacts ([dba1c92](https://github.com/Crimsab/oneday/commit/dba1c9220ee9d19c92c8d941b17655e9887df7ba))
* **tui:** color slash commands in input ([5fc6592](https://github.com/Crimsab/oneday/commit/5fc6592b73d03645e8535b3dbfab96aa3b291082))
* **tui:** refresh live slash command styling ([3438712](https://github.com/Crimsab/oneday/commit/343871235fd21468481a360be50e24fb16b6741b))

## [1.3.0](https://github.com/Crimsab/oneday/compare/v1.2.0...v1.3.0) (2026-04-09)


### Features

* **ai:** complete phase 10 ambient ascii benchmarking ([accba8d](https://github.com/Crimsab/oneday/commit/accba8dc4a25becc3029d012d1b35c755706df69))

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
