# Changelog

## [1.17.0](https://github.com/Crimsab/oneday/compare/v1.16.0...v1.17.0) (2026-07-31)


### Features

* **desktop:** add optional managed Codex setup ([779b2d1](https://github.com/Crimsab/oneday/commit/779b2d1bfee609a30e5324295709f2edac68901b))


### Bug Fixes

* **release:** authenticate container verification ([f43acbf](https://github.com/Crimsab/oneday/commit/f43acbf871ff237f53b464f7879dafde0fa12147))
* **release:** make sealed asset reruns idempotent ([cd92037](https://github.com/Crimsab/oneday/commit/cd92037482cd7ba3599424965ace3a24df733273))
* **release:** normalize image references and clarify OAuth ([fae39aa](https://github.com/Crimsab/oneday/commit/fae39aa53d66cfd99a3d5b36faa23b96fc0c1803))
* **release:** use current publisher on reruns ([b860ace](https://github.com/Crimsab/oneday/commit/b860aceaf4765fbee1e1f33fad9b066ff2387169))

## [1.16.0](https://github.com/Crimsab/oneday/compare/v1.15.0...v1.16.0) (2026-07-30)


### Features

* add portable Docker bootstrap and macOS releases ([92c81ad](https://github.com/Crimsab/oneday/commit/92c81ad65dfb01b2b407dbaa40acf7156fbd08e2))
* **cli:** add optional media setup onboarding ([bc9c264](https://github.com/Crimsab/oneday/commit/bc9c2643e5a687588724daf14d6f1cae8e48ade7))
* **desktop:** add standalone launch profiles ([fe986b0](https://github.com/Crimsab/oneday/commit/fe986b08f3003d868fa219efad3f4bb131ac1f14))
* **discovery:** cache provider catalogs and organize AI settings ([310c04d](https://github.com/Crimsab/oneday/commit/310c04d91752edc2160951bbcfce18392f046e1c))
* **imagegen:** add optional bridge compose profile ([ba75805](https://github.com/Crimsab/oneday/commit/ba75805f4ff4de2bcc58c3ba9eacfe5a724113a9))
* **imagegen:** harden compatible media providers ([787ca99](https://github.com/Crimsab/oneday/commit/787ca99f459de6c06c7f085986e7b3a43d79a9a3))
* improve visual asset inspection and annotations ([19d6a57](https://github.com/Crimsab/oneday/commit/19d6a57e70ebba944cca947d6384356296c19402))
* **media:** add studio filters and bridge fallback ([42c0ccb](https://github.com/Crimsab/oneday/commit/42c0ccb418bd419b75606e2c16751fba9a6f8b09))
* **media:** preserve canonical visual continuity context ([5a64451](https://github.com/Crimsab/oneday/commit/5a64451973538b2bdf16853ca8d5713b5bf4dd6e))
* **models:** discover configured provider catalogs ([4fdc266](https://github.com/Crimsab/oneday/commit/4fdc266a836461df5f0604e9015fb8e1aadd9542))
* **recovery:** add redacted diagnostics ([a79afa5](https://github.com/Crimsab/oneday/commit/a79afa5282f1a2fd96705af72c7c2ea15f93743a))
* **setup:** add readiness diagnostics ([a5814be](https://github.com/Crimsab/oneday/commit/a5814be3d1ec0581ae16b45cc81e72f1f0939e8f))
* **web:** add installation readiness onboarding ([7fa7b55](https://github.com/Crimsab/oneday/commit/7fa7b5552807f674610951d5420876ce3730ebd1))
* **web:** redesign history timeline ([68a9948](https://github.com/Crimsab/oneday/commit/68a9948ee3bcd7c2df0bbb9f31960186c0dab34d))
* **web:** redesign visual asset workspace ([b6ca6ab](https://github.com/Crimsab/oneday/commit/b6ca6aba362276fcb9fe65bd0f92ce9da48c731c))
* **web:** refine setup and install shell ([3979a1f](https://github.com/Crimsab/oneday/commit/3979a1fc4c14149936668f05c3dea71912e35d07))
* **web:** reshape installation setup console ([ec60605](https://github.com/Crimsab/oneday/commit/ec606056b009c278b099fd4b5530562d6381d969))
* **web:** separate player and operator settings ([bc99443](https://github.com/Crimsab/oneday/commit/bc99443baddd819b30f87638f38ec2c83673fcd9))
* **web:** wire history timeline actions ([77d2678](https://github.com/Crimsab/oneday/commit/77d267835846f9ceb88582ffaff8dd01936a24c8))


### Bug Fixes

* allow internal Homepage health checks ([f8cd012](https://github.com/Crimsab/oneday/commit/f8cd01283e8262583a408574f22fa4086bc8e2a9))
* **cli:** keep media credentials out of yaml ([8989978](https://github.com/Crimsab/oneday/commit/8989978f6fd3c2eee37086afda0408f71ce37017))
* **cli:** preserve media setup and credential boundaries ([c9e6815](https://github.com/Crimsab/oneday/commit/c9e6815adf29b544a40de5d4d0cfbc6accd066d8))
* **cli:** roll back env on config write failure ([2944959](https://github.com/Crimsab/oneday/commit/2944959b81c09969031399943c392eae25da52d2))
* compact visual version details ([c6bf189](https://github.com/Crimsab/oneday/commit/c6bf189b33e6e33336c966b448fba0402f77ba59))
* **deploy:** pass gateway auth credentials through compose ([afe589c](https://github.com/Crimsab/oneday/commit/afe589cba5873440f241b487b6680ddb928718e0))
* **desktop:** harden standalone lifecycle ([c34a75a](https://github.com/Crimsab/oneday/commit/c34a75aa53573df28658e355d35ab50756761fef))
* **desktop:** make Windows job containment compile ([7c237e8](https://github.com/Crimsab/oneday/commit/7c237e8af09f447f157b50a67f4968f4b58620f1))
* **desktop:** package and await standalone resources ([4555c37](https://github.com/Crimsab/oneday/commit/4555c37f2c8f38d089becfa33b847aba4e8d84c0))
* **desktop:** refine profile action typography ([3a32591](https://github.com/Crimsab/oneday/commit/3a32591877fb9c28af8fb7752bc9fa6f660dd621))
* **desktop:** terminate Linux sidecars with parent ([455f492](https://github.com/Crimsab/oneday/commit/455f49219f4031f19bfeee26f89913de38830933))
* **discovery:** validate provider catalog integration ([0240026](https://github.com/Crimsab/oneday/commit/02400264fbece4877df15daf551ffe9d9f14c449))
* **gateway:** align OpenTelemetry dependencies ([3ca08fd](https://github.com/Crimsab/oneday/commit/3ca08fd183bfb911b45daf3aa1dbf7c13ef8222a))
* **gateway:** close auth boundary gaps ([6b7277d](https://github.com/Crimsab/oneday/commit/6b7277da6ce51375bd335b56419c61d226a0b1af))
* **gateway:** close standalone auth gaps ([2b52f57](https://github.com/Crimsab/oneday/commit/2b52f578f5ac8f5c29993bc01f38162f6512debc))
* **gateway:** preserve ambiguous image job outcomes ([673380a](https://github.com/Crimsab/oneday/commit/673380aa5d9e31ba6635b238ee54bfb940839fd3))
* **gateway:** secure local API access ([0824d53](https://github.com/Crimsab/oneday/commit/0824d533f5417e701a2df4c8dfcdea1df22b6c00))
* harden first-run recovery and visual workspace ([971f41d](https://github.com/Crimsab/oneday/commit/971f41d8c22991e3e80de20d5680e5fdcefedd8a))
* **imagegen:** restrict local no-auth providers ([e93b404](https://github.com/Crimsab/oneday/commit/e93b40411608952e510d04ff1ce11d62a5f43c57))
* make docs compose link portable ([61868db](https://github.com/Crimsab/oneday/commit/61868db59ecf1cb52ee66ec288fa34a0fb785283))
* make public Docker setup self-contained ([606c48f](https://github.com/Crimsab/oneday/commit/606c48f1b5bcffa4374f6b4f4955fb771fd6e648))
* make release metadata portable on macOS ([6a503fa](https://github.com/Crimsab/oneday/commit/6a503fa60b997c948debf7d1803797bd9b61e5a5))
* **media:** compile visual continuity metadata ([63d5320](https://github.com/Crimsab/oneday/commit/63d5320d6db88787abb96181c6d620bb44a9e02e))
* publish portable backup checksums ([4a0d266](https://github.com/Crimsab/oneday/commit/4a0d266cc7530df8bf3a685f3048ec21f311a0eb))
* **recovery:** preserve backup on checksum collision ([d0c1df2](https://github.com/Crimsab/oneday/commit/d0c1df262b9d11ebed0088c242083e13e71c78ac))
* **recovery:** prevent backup publication clobber ([163aef0](https://github.com/Crimsab/oneday/commit/163aef0187fd5eed0bfe50420e0955553d6efe3c))
* **recovery:** rollback checksum publication race ([a7e3e80](https://github.com/Crimsab/oneday/commit/a7e3e80917b5012ea8f144185a69800f15e22f18))
* **release:** harden package publication ([dd27b30](https://github.com/Crimsab/oneday/commit/dd27b30d5ab3f83c315e8d3cecf65d9a1db7ac14))
* restore durable browser authentication ([c9bba7e](https://github.com/Crimsab/oneday/commit/c9bba7ec7d45a0bd76a00a44cdf0b4b3ae27083b))
* **setup:** honor gateway database path ([5ee8961](https://github.com/Crimsab/oneday/commit/5ee896125cda58bc844026e92ec51c43753a4464))
* **setup:** probe local gateway readiness ([ef2b141](https://github.com/Crimsab/oneday/commit/ef2b141ede91d6a22117a6dcda432d623888b959))
* **setup:** redact diagnostic paths and colocate dotenv ([a12542e](https://github.com/Crimsab/oneday/commit/a12542e329d4c465cb3f7502da52c244e9a7afd5))
* **web:** align media controls and PWA actions ([b49b39f](https://github.com/Crimsab/oneday/commit/b49b39ff1a1f2ee8da12df2a0bb1a3e130f490b7))
* **web:** align settings search scopes ([bd12be8](https://github.com/Crimsab/oneday/commit/bd12be836830cad47bb906b735429f109e1c733c))
* **web:** clarify visual workspace labels ([7ad0185](https://github.com/Crimsab/oneday/commit/7ad018535ccb80510da09758002aba087db3a832))
* **web:** compact story toolbar and PWA notices ([ce39da4](https://github.com/Crimsab/oneday/commit/ce39da4e7ac62954bd567ebe0dace0b07a119174))
* **web:** improve onboarding accessibility ([3f9cd5e](https://github.com/Crimsab/oneday/commit/3f9cd5e47c4fd1b69ccba6e06475b5b6e6dda007))
* **web:** keep media inspector control visible ([6892431](https://github.com/Crimsab/oneday/commit/68924311058d905cafa40fbdfe3fa8eca506aeb0))
* **web:** keep setup bootstrap instance-safe ([cdf4026](https://github.com/Crimsab/oneday/commit/cdf4026eb96dbb89f25c1040c28398f3ab4e7aa0))
* **web:** keep visual preview proportionate ([161b11a](https://github.com/Crimsab/oneday/commit/161b11a8b2ebca94853c048496533701da46d2ec))
* **web:** let PWA updates claim active tabs ([95802e9](https://github.com/Crimsab/oneday/commit/95802e9376e650d2898245098d881f0760d9d8b0))
* **web:** make visual editor a contained dialog ([91b6158](https://github.com/Crimsab/oneday/commit/91b6158c5aeba2b034349062fee689a076febfbc))
* **web:** open media editor as side sheet ([ab7d19c](https://github.com/Crimsab/oneday/commit/ab7d19ccef0906db7bb43b2528b6c251fde271f2))
* **web:** preserve media library context under sheet ([445cfba](https://github.com/Crimsab/oneday/commit/445cfba31448d9be08266828755476298a7520df))
* **web:** refine media studio and settings surfaces ([c5833f0](https://github.com/Crimsab/oneday/commit/c5833f02cb2b258af5bbf4f042ecdcb2182d2719))
* **web:** refine operator console and compact history ([94fbfc9](https://github.com/Crimsab/oneday/commit/94fbfc98abf927c3abe6778ea3ff78184879e43f))
* **web:** reliably activate PWA updates ([d9a23cc](https://github.com/Crimsab/oneday/commit/d9a23cc5b4083ab2954cd9b1950fd68ad88ad0dc))
* **web:** restore story library labels and toolbar layout ([4ef6a6f](https://github.com/Crimsab/oneday/commit/4ef6a6fc9e7205f860e36c1692d2328bebc3e5cf))
* **web:** stack media sheet content correctly ([0c2b8a2](https://github.com/Crimsab/oneday/commit/0c2b8a2c6fea6abbc03b3769058ab264915caef1))
* **web:** unify setup and media editing ([270ba34](https://github.com/Crimsab/oneday/commit/270ba343ec18e04849ff0f10b52a3c8c238dad01))

## [1.15.0](https://github.com/Crimsab/oneday/compare/v1.14.0...v1.15.0) (2026-07-16)


### Features

* add persistent translation and story portability ([7a580f4](https://github.com/Crimsab/oneday/commit/7a580f45c3d88188d4be8ae3af43223e72989f66))
* complete story details, custom assets, and theme bundles ([55888ae](https://github.com/Crimsab/oneday/commit/55888ae36321784e092fb67316b484ad20d8355e))
* **desktop:** add secure Tauri remote client ([43ac900](https://github.com/Crimsab/oneday/commit/43ac900d9801d34be2881a0409548faa1acf2002))
* **web:** add canonical story routes ([8ef5957](https://github.com/Crimsab/oneday/commit/8ef5957ffd10de09c6de1ac9324beaf6b3da3217))
* **web:** add server-connected PWA ([753d525](https://github.com/Crimsab/oneday/commit/753d5257db82d53f59aa4cbd45f91e8982bff2be))
* **web:** add story library, uploads, themes, and translation ([922a2b6](https://github.com/Crimsab/oneday/commit/922a2b6a7bfbcf0d1fee2fba5ab2d9e06a1a1166))
* **web:** expose story exports in the library ([8a8900c](https://github.com/Crimsab/oneday/commit/8a8900cd3d9f8ce1bcea5bc82008d977fbb69889))


### Bug Fixes

* **build:** include PWA config in gateway image ([5e2ad4a](https://github.com/Crimsab/oneday/commit/5e2ad4a55954abde0a8828df33037ccd34ea5476))
* **gateway:** serve SPA routes with success status ([5218cf2](https://github.com/Crimsab/oneday/commit/5218cf2b1a0ed31dd434146cd4bed63e3b0ba015))
* **web:** align translation controls with their engine ([9f29b0b](https://github.com/Crimsab/oneday/commit/9f29b0b8b2db55f7c2bb12863c56c3337d4d6fa9))
* **web:** guard lazy story detail transitions ([c9a354f](https://github.com/Crimsab/oneday/commit/c9a354f6729aeecee4fd7f6e8659a3b9059d88cf))
* **web:** harden translation center responses ([c842faa](https://github.com/Crimsab/oneday/commit/c842faa3b829e11cb470921116ae739432df4053))
* **web:** integrate the collapsed story count badge ([9d4720e](https://github.com/Crimsab/oneday/commit/9d4720e0e62df3d3a53b52ad4d0f885ab2304ae0))
* **web:** keep language picker inside viewport ([0f706b7](https://github.com/Crimsab/oneday/commit/0f706b70c2bf4d9dbe5dff3cfab86558881c3d98))
* **web:** polish story portability controls ([6bd4433](https://github.com/Crimsab/oneday/commit/6bd44339d8a4c20656663482c180cd87806d27bf))
* **web:** repair collapsed rail branding and layout ([47e686b](https://github.com/Crimsab/oneday/commit/47e686b8e7c3a54913948b506f45cfdd81b56506))
* **web:** size story library content rows ([c094dd7](https://github.com/Crimsab/oneday/commit/c094dd7612d20f653c71b6d1ac5df6ca3cf490c3))

## [1.14.0](https://github.com/Crimsab/oneday/compare/v1.13.0...v1.14.0) (2026-07-15)


### Features

* **imagegen:** add native image editing operations ([493611d](https://github.com/Crimsab/oneday/commit/493611d7fc6774c8bc018ad2841015734fdb7afd))
* **web:** add native image mask editor ([9ec181c](https://github.com/Crimsab/oneday/commit/9ec181cea82eb15b0528e71e5c674bb5fa6ba217))


### Bug Fixes

* **imagegen:** recover interrupted edit jobs safely ([5b790f5](https://github.com/Crimsab/oneday/commit/5b790f59dff62b414c1d25571274a9cfe37f1a52))

## [1.13.0](https://github.com/Crimsab/oneday/compare/v1.12.0...v1.13.0) (2026-07-15)


### Features

* **web:** finish personalized options and support tools ([15c44fe](https://github.com/Crimsab/oneday/commit/15c44fe5167c810fb29e36a248092ea892607e74))


### Bug Fixes

* **gateway:** forward excluded minigame families ([87b3cd6](https://github.com/Crimsab/oneday/commit/87b3cd6ef783488ff2cdc2e56b850e08d99ef754))

## [1.12.0](https://github.com/Crimsab/oneday/compare/v1.11.0...v1.12.0) (2026-07-15)


### Features

* **web:** complete personalized settings ([920bb79](https://github.com/Crimsab/oneday/commit/920bb79e3eb5c3df68165427d8acfe264ec77315))


### Bug Fixes

* **release:** embed build metadata in container ([1945271](https://github.com/Crimsab/oneday/commit/1945271ece78f1935e00ba6c49f13d6a2a0b68b9))

## [1.11.0](https://github.com/Crimsab/oneday/compare/v1.10.0...v1.11.0) (2026-07-15)


### Features

* **web:** personalize options workspace ([0ad3637](https://github.com/Crimsab/oneday/commit/0ad3637decad856abbf2fef27109aad75857e460))


### Bug Fixes

* **web:** improve player accessibility ([7f76ba7](https://github.com/Crimsab/oneday/commit/7f76ba7a46240e56f97d33a364bb30f42f9019f6))

## [1.10.0](https://github.com/Crimsab/oneday/compare/v1.9.0...v1.10.0) (2026-07-15)


### Features

* **brand:** adopt the OneDay app icon ([47da14f](https://github.com/Crimsab/oneday/commit/47da14f3b67d55c445c27ca4b8aa5f84533bbb33))
* **i18n:** add English and Italian interfaces ([866b1c9](https://github.com/Crimsab/oneday/commit/866b1c92943d1d97389d84603b45ec901c4e7560))
* **i18n:** add English and Italian interfaces ([2227b17](https://github.com/Crimsab/oneday/commit/2227b177355c89fd0c4fc794f6e5ce5203a260b2))
* **imagegen:** add explicit provider adapters ([40c922d](https://github.com/Crimsab/oneday/commit/40c922d2cc271c9f4ab9cc948cec89cfc10d1304))
* prepare OneDay for public documentation ([91b2879](https://github.com/Crimsab/oneday/commit/91b2879ef368d49ebbd624ae6e59b1756ba6176f))
* **settings:** redesign image provider configuration ([87a09fa](https://github.com/Crimsab/oneday/commit/87a09fa3559efaeea0562faea413be6b2f48b486))


### Bug Fixes

* **ci:** install required workflow runtimes ([a742753](https://github.com/Crimsab/oneday/commit/a7427535763cb7f184258e63c28e2b654eb26661))
* **ci:** isolate runner temporary files ([1e92abe](https://github.com/Crimsab/oneday/commit/1e92abe8240f606005a879e3ca0a56c7ad7fbecc))
* **ci:** keep temporary files off slow runner storage ([ff721f2](https://github.com/Crimsab/oneday/commit/ff721f21ae2d266f39bb2da56656eb718d0e7ada))
* **imagegen:** serialize capability arrays consistently ([2bf8543](https://github.com/Crimsab/oneday/commit/2bf85438d4d0db17cda8cb28c5e4b1b650081bac))
* **release:** isolate runner temporary files ([03e5139](https://github.com/Crimsab/oneday/commit/03e5139721323cf681471d6046c1f34fbcae9557))

## [1.9.0](https://github.com/Crimsab/oneday/compare/v1.8.0...v1.9.0) (2026-07-15)


### Features

* **imagegen:** default to Codex Responses routing ([9080ea8](https://github.com/Crimsab/oneday/commit/9080ea829ce9ec10a80debbcda3b4d44c50bbe62))
* **imagegen:** integrate native bridge routing ([c91f4f3](https://github.com/Crimsab/oneday/commit/c91f4f3c9eb95427984c2b989290d6b18ba2b686))
* **observability:** add optional OTLP tracing ([e0e176c](https://github.com/Crimsab/oneday/commit/e0e176c0e38ba19f50a8a1bd84c726b1ec5aa4fd))

## [1.8.0](https://github.com/Crimsab/oneday/compare/v1.7.0...v1.8.0) (2026-07-14)


### Features

* add accessible minigame selection ([34149c8](https://github.com/Crimsab/oneday/commit/34149c84eb6413039e3029f26e3e98cf6c5b96fa))
* add branch-safe audio controls ([810cf0d](https://github.com/Crimsab/oneday/commit/810cf0d253e45356971300f2000375ec805c8e6a))
* add branch-safe visual version controls ([a87dc9c](https://github.com/Crimsab/oneday/commit/a87dc9cf7926ab173abc60293d8c0de0996257e4))
* add canonical audio storage ([14f65ef](https://github.com/Crimsab/oneday/commit/14f65ef1111b21b1849cc14736bc76b69d704c53))
* add canonical world time weather and travel ([a71303f](https://github.com/Crimsab/oneday/commit/a71303f7cbae1b85a0e11b6aa97ccc006cc6004c))
* add causal generation telemetry schema ([971306b](https://github.com/Crimsab/oneday/commit/971306bc3e9820e4991f2e27a0050cc82f66998f))
* add cross-surface minigame families ([4bf611e](https://github.com/Crimsab/oneday/commit/4bf611e395c24e92fe05bb786ef21506dd78bc05))
* add hierarchical canonical world maps ([66cc2b7](https://github.com/Crimsab/oneday/commit/66cc2b7ef493f2b9a57b9738a93cbeae93ac3e7d))
* add illustrated canonical map layers ([b43d2ee](https://github.com/Crimsab/oneday/commit/b43d2eedc7ac3be675547cbad2115aababf90a92))
* add minigame authoring and fairness evals ([dcf1e9e](https://github.com/Crimsab/oneday/commit/dcf1e9e1bf191c31e9639e54b6dd244fc56beb52))
* add replayable minigame host ([afd1718](https://github.com/Crimsab/oneday/commit/afd171824006ed1f428d7b4992cedc59e356177a))
* add universal simulation authoring and exports ([1914629](https://github.com/Crimsab/oneday/commit/1914629f2be280a764d3b2bb8f5b98d72fcb3f0b))
* adopt carbon theme and recover visual assets ([002a06c](https://github.com/Crimsab/oneday/commit/002a06c8de9dc459af186d456293eda9bef37a55))
* **bridge:** emit versioned typed errors ([e53e8f9](https://github.com/Crimsab/oneday/commit/e53e8f9be504b242013fdede76fafdeb2312658a))
* complete canonical audio lifecycle ([bdd9dba](https://github.com/Crimsab/oneday/commit/bdd9dba3a76054304097a35fd3b44e677029818d))
* establish transactional canon and world foundations ([4149d68](https://github.com/Crimsab/oneday/commit/4149d68d604f329dafc04821506b3ec7b54cacd8))
* expose redacted generation diagnostics ([e53c4cb](https://github.com/Crimsab/oneday/commit/e53c4cbdee0fdaeae0350e117cbd43bae98bfa90))
* gate images on canonical branch state ([28710c9](https://github.com/Crimsab/oneday/commit/28710c93c91e6fdfd29f948d954e100077ab7149))
* **gateway:** add request tracing ([967d0ce](https://github.com/Crimsab/oneday/commit/967d0ce79806602f0be8d541052f83952b9b8ab0))
* **imagegen:** route transparent map icons separately ([55dfc96](https://github.com/Crimsab/oneday/commit/55dfc96b939572d4b1fe5e3c367cf61424724c2d))
* make known map interactive ([8ba2253](https://github.com/Crimsab/oneday/commit/8ba2253901847b29c1f1ac5256d58e6cbcea50d2))
* make outcomes authoritative and replayable ([580c9f9](https://github.com/Crimsab/oneday/commit/580c9f984f58d3e3916c591d483f39b10108f416))
* **observability:** propagate HTTP request IDs ([be4623e](https://github.com/Crimsab/oneday/commit/be4623e04c06fa90b289064dbb007e649f0f2632))
* organize options into searchable settings workspace ([2bd3b44](https://github.com/Crimsab/oneday/commit/2bd3b44927264061a005adc7c621f11f9bd35109))
* queue canonical text for speech ([05f968d](https://github.com/Crimsab/oneday/commit/05f968d9bb8af79d19dc8334955ca4ef294d3f8b))
* redesign browser around narrative play ([670156c](https://github.com/Crimsab/oneday/commit/670156c3cb423b77c18b4c30f1f9b0ae27032a31))
* refine narrative choices and brand identity ([d5ca5b3](https://github.com/Crimsab/oneday/commit/d5ca5b33a46f24971aa975c1dbd7150426058492))
* refine narrative controls and visual workflow ([535a040](https://github.com/Crimsab/oneday/commit/535a040be5c45767b8ddb0c472215c1a292e3697))
* **release:** publish versioned GHCR images ([f0cfeb3](https://github.com/Crimsab/oneday/commit/f0cfeb3b0d15f22e6135349fc8fe5627b6604c25))
* trace causal AI generation stages ([de48a92](https://github.com/Crimsab/oneday/commit/de48a929153e4f7326f51f8eebfb6f82af2c260d))
* trigger minigames automatically from narrative ([a335ed2](https://github.com/Crimsab/oneday/commit/a335ed2ab016a11ea439bada71160da64d44ce2c))
* unify branch navigation and narrative surfaces ([f19ebaa](https://github.com/Crimsab/oneday/commit/f19ebaa2f15e8971c3b413cb2f85ead4c9307401))
* unify story modules and crafting workspace ([f7d03f9](https://github.com/Crimsab/oneday/commit/f7d03f94b9fa3855dbb5ea5721c17c0b9f9da2a3))
* version visual canon by branch lineage ([14aeee5](https://github.com/Crimsab/oneday/commit/14aeee5bba6090bcdda8d9e602711a5414f590e6))
* **web:** choose visual style during story creation ([1438ea6](https://github.com/Crimsab/oneday/commit/1438ea67f5c2398baee7e0a881a0121ef5b2c7e1))


### Bug Fixes

* add review-first story presets ([ee6f148](https://github.com/Crimsab/oneday/commit/ee6f148ccbb393a588af6407de4ee21c91df5b91))
* **ai:** bound codex subprocess output ([468cee9](https://github.com/Crimsab/oneday/commit/468cee97f246125ff51b2a9fb9741a6c22712181))
* **ai:** bound provider response bodies ([870952e](https://github.com/Crimsab/oneday/commit/870952e650caba5a45c9c607454f6500002a3f77))
* align choice detail preference ([5a1b8e4](https://github.com/Crimsab/oneday/commit/5a1b8e4fdce53b950019ded158d3bc19cde55eed))
* align narrative controls and contextual history ([2a466a1](https://github.com/Crimsab/oneday/commit/2a466a178908d36c5bc1467efbb1880b768407c0))
* **benchmarks:** bound provider response bodies ([6d1a86f](https://github.com/Crimsab/oneday/commit/6d1a86f3682455ed3ce966146dbc62e9471d28f5))
* **bridge:** enforce ordered stream frames ([e97f8e5](https://github.com/Crimsab/oneday/commit/e97f8e5a4a059e2f8af862f93350c658d20ee187))
* **bridge:** preserve load snapshot metadata ([8fe0c0e](https://github.com/Crimsab/oneday/commit/8fe0c0e0ff14d12ce60f56493fc1b6ae89556474))
* **bridge:** preserve typed gateway errors ([b511886](https://github.com/Crimsab/oneday/commit/b5118860a209458ae855380350780af28ad078cb))
* build main artifact in release gate ([40de21f](https://github.com/Crimsab/oneday/commit/40de21fb795ccb4b570cea208d9dac4ab8f628c3))
* **build:** include generated gateway contracts ([1037311](https://github.com/Crimsab/oneday/commit/10373111a5bff4f294605cb34137524dcf46ced7))
* **build:** pin security-patched Go toolchain ([86a2378](https://github.com/Crimsab/oneday/commit/86a23783f36854ffafd3886617241c27e50714f8))
* bundle story engine in gateway image ([7c6d2d8](https://github.com/Crimsab/oneday/commit/7c6d2d89e8c5aec0384bb8bfae08e63fc405720d))
* **ci:** harden workflows for public contributions ([ef1801c](https://github.com/Crimsab/oneday/commit/ef1801cde8c01719f22bf06db5edeba448430ae1))
* **ci:** route private runs to the self-hosted runner ([ef19af6](https://github.com/Crimsab/oneday/commit/ef19af684114d679fcb7779ca7f244329cdb35dd))
* clarify map generation readiness ([c196313](https://github.com/Crimsab/oneday/commit/c19631316561c74688af413e2c18e875999197f1))
* complete mobile module and asset workflows ([6c6055a](https://github.com/Crimsab/oneday/commit/6c6055a3a52b75d0eaf612a7e193d229d4a73cd2))
* **config:** replace private-network-specific defaults ([beca80b](https://github.com/Crimsab/oneday/commit/beca80b72c2e86f00127f1f31abdb68b5294b259))
* **db:** scope NPC facts to active lineage ([f89e967](https://github.com/Crimsab/oneday/commit/f89e96748a360c101340054316f292376e913c8c))
* **deps:** update goldmark for markdown safety ([9bfa1bc](https://github.com/Crimsab/oneday/commit/9bfa1bc819116bd0ea2a6c703d141d17ec37de8b))
* **docker:** publish a portable compose stack ([d4e64b1](https://github.com/Crimsab/oneday/commit/d4e64b1d8db3e1b4e78380dd914ed9b05d1105bf))
* **docker:** require an explicit config file ([cced045](https://github.com/Crimsab/oneday/commit/cced045fed5d19d0d3c89924c1b94e03386bef87))
* **engine:** commit crafting outcomes atomically ([007da5b](https://github.com/Crimsab/oneday/commit/007da5b00f5ac5d25a20f08837cdcab43d0eff08))
* **engine:** commit narrator meta changes atomically ([fbd100a](https://github.com/Crimsab/oneday/commit/fbd100abc189350d4ad030618bd9aaf2e3062a57))
* **engine:** finalize combat outcomes atomically ([71cd1d0](https://github.com/Crimsab/oneday/commit/71cd1d0fbaf45a74921845281e198431dffd8678))
* **engine:** order state changes deterministically ([4c9df90](https://github.com/Crimsab/oneday/commit/4c9df90f97e97b428bf194f8ac3d86098116b865))
* **engine:** propagate social duel persistence errors ([b5f93dd](https://github.com/Crimsab/oneday/commit/b5f93dd31692e71ea9805bba778803d388fe6723))
* **engine:** reject malformed numeric state changes ([b0480bf](https://github.com/Crimsab/oneday/commit/b0480bf9aceb20c8c8f27376582ebea0166e2eb0))
* **engine:** rollback failed combat turns ([d008b16](https://github.com/Crimsab/oneday/commit/d008b160353bdc8f234f99ffde9ade6c6fde78c6))
* **gateway:** enforce bridge process-group timeouts ([9b59a80](https://github.com/Crimsab/oneday/commit/9b59a80daa1b3c419e33d5958deb6d8b1ae98277))
* **gateway:** initialize fresh browser databases ([23259fd](https://github.com/Crimsab/oneday/commit/23259fd6b3563fc8a4e107912be65140c06b0753))
* **gateway:** make story snapshots transactionally consistent ([d1a7457](https://github.com/Crimsab/oneday/commit/d1a74573c061471e0471abc66cd555959c217ad2))
* **gateway:** make visual asset transitions atomic ([e390c34](https://github.com/Crimsab/oneday/commit/e390c3443e3c638db62702d93e0615201bffb0c9))
* **gateway:** preserve bridge transport errors ([075b31f](https://github.com/Crimsab/oneday/commit/075b31f7a023b508cb8ab0fcd8b10b5f5a0913c0))
* **gateway:** stream bounded audio assets ([da33895](https://github.com/Crimsab/oneday/commit/da338950d7c83a4cf30156818aa43d4560ce4bb7))
* **gateway:** terminate timed-out bridge processes ([5b7e7a3](https://github.com/Crimsab/oneday/commit/5b7e7a39da306e8cbfe2718816eaedf08eb350e5))
* **history:** debounce and cancel branch searches ([548346e](https://github.com/Crimsab/oneday/commit/548346ed4035c592e529e5b4cca6acf7b1f3d395))
* **idempotency:** bound replay retention ([15fbddb](https://github.com/Crimsab/oneday/commit/15fbddb9dca1bb4af02c148a46a40a06ac425844))
* ignore playwright run artifacts ([801c6a6](https://github.com/Crimsab/oneday/commit/801c6a6db0b517bf012b507fcb0087d2235a1634))
* **imagegen:** route the private bridge through its configured network ([ad34713](https://github.com/Crimsab/oneday/commit/ad3471361e729da4ef90d592d3a7c70978255b46))
* **logging:** surface successful bridge diagnostics ([02cb09b](https://github.com/Crimsab/oneday/commit/02cb09b35810804556f78256013f82c717d4ed21))
* make known map interactions spatial ([21170c0](https://github.com/Crimsab/oneday/commit/21170c0c724bd47ed86e890c35b042f3e9c39ec0))
* normalize story library spacing ([fcae066](https://github.com/Crimsab/oneday/commit/fcae066bf1e8cf218d0656080ce78352cb5903bb))
* preserve focus and panel accessibility ([6f1be64](https://github.com/Crimsab/oneday/commit/6f1be6441471f37dbf98f199bc78d426f1450803))
* **rag:** bound and coalesce background work ([36d1955](https://github.com/Crimsab/oneday/commit/36d1955a3f70d9bdfbdffcffa34b5753fc56fc98))
* recover timeline and align branch controls ([896424e](https://github.com/Crimsab/oneday/commit/896424e04818da77c38f8e9d051069abcd8faad2))
* **release:** anchor the current release lineage ([db7a0ba](https://github.com/Crimsab/oneday/commit/db7a0bae583ca27d3213ae2cb4bd83a724b2a18f))
* **release:** restore changelog automation state ([27bb560](https://github.com/Crimsab/oneday/commit/27bb5608562932d5f23d196d1ec7e27f94d50ff1))
* **release:** target repository when dispatching CI ([3fb999d](https://github.com/Crimsab/oneday/commit/3fb999db82bbe187b11bf57b8f84461dff1ebc6e))
* **release:** validate automated release pull requests ([95dcac5](https://github.com/Crimsab/oneday/commit/95dcac5fdcaa6287a43a1c78f7800138519d6833))
* restyle history message cards ([1bbbd19](https://github.com/Crimsab/oneday/commit/1bbbd19a710457387e43f60649622cd56fa9f5bc))
* route image generation across hosts ([a9daef3](https://github.com/Crimsab/oneday/commit/a9daef3a24355dc87a5851c4ea39223f3d5cc991))
* satisfy Rust 1.97 Clippy ([c34da1c](https://github.com/Crimsab/oneday/commit/c34da1cdd6128dcdf2ef8710c403b8993e29e386))
* scope branch navigation to message decisions ([0ff295b](https://github.com/Crimsab/oneday/commit/0ff295b84f36a1565ea367110349513db97c2b2f))
* separate choice rollback from path pager ([af9f1b7](https://github.com/Crimsab/oneday/commit/af9f1b7929514c79210fac457963e397984b9f5a))
* **setup:** report env file creation failures ([4a5c651](https://github.com/Crimsab/oneday/commit/4a5c6510bad5ede77394efa4bfd8d132af99cf55))
* stabilize visual asset lineage and versions ([2a74c17](https://github.com/Crimsab/oneday/commit/2a74c179f84468324d40e4c8dbbc3eb746254733))
* **streaming:** release pipelines on cancellation ([2c1b87c](https://github.com/Crimsab/oneday/commit/2c1b87cbbac4c4ef7980f03bef8078126b9eed1e))
* streamline turn feedback and navigation ([e3761e2](https://github.com/Crimsab/oneday/commit/e3761e2e6d1bb3e9c1a8d3e400cdb7bfdeb0fb5d))
* **telemetry:** surface image trace failures ([d4f52a9](https://github.com/Crimsab/oneday/commit/d4f52a9ea2416d0a3d9a205939a9854023e8e70f))
* tighten story player interaction surfaces ([efe9471](https://github.com/Crimsab/oneday/commit/efe9471f6e938e282a3c45e5dcc6d76491cc5da3))
* **timeline:** atomically fork and restore decisions ([71e7112](https://github.com/Crimsab/oneday/commit/71e71125249c42f8cc599183cfcf3ea2e3394ed3))
* unify selected borders and history palette ([3ac24a6](https://github.com/Crimsab/oneday/commit/3ac24a6d87e9aa0e1748a2a62b55371bebb22e3c))
* **web:** bound API requests with abort timeouts ([15be052](https://github.com/Crimsab/oneday/commit/15be05260bcedb876a420f52bc75624d9cd96ddf))
* **web:** make modal focus keyboard-safe ([5241127](https://github.com/Crimsab/oneday/commit/52411274d1f6e70b44eb30789babefa97bd30664))
* **web:** normalize empty audio responses ([83556df](https://github.com/Crimsab/oneday/commit/83556dfb078d6cd8cbc6e3c12aa22914f46de702))
* **web:** preserve transcript reading position ([92591ce](https://github.com/Crimsab/oneday/commit/92591ce2b8253c2aeffd36799247ee4c3fe20e8b))
* **web:** prevent stale story effects ([2b5f979](https://github.com/Crimsab/oneday/commit/2b5f979efbf6f4fba1ae42bebe06106df8a50fdf))
* **web:** respect speech mode in deferred audio controls ([c363494](https://github.com/Crimsab/oneday/commit/c363494b8e450850a585465f179b2693a71651fa))
* **web:** serialize message audio autoplay ([93c77e7](https://github.com/Crimsab/oneday/commit/93c77e731f3343feffd26a3a39578b42d1f428a9))


### Performance Improvements

* **audio:** bound unreferenced TTS cache retention ([25765c1](https://github.com/Crimsab/oneday/commit/25765c1a8a5e5b9a0ed3c48f21f8e9f7ef285787))
* **audio:** reuse pronunciation lexicons per message ([da9ded4](https://github.com/Crimsab/oneday/commit/da9ded4b42754f2824483cc35835377bc6a47f49))
* **export:** render archives off async workers ([718d8e0](https://github.com/Crimsab/oneday/commit/718d8e0c74b94072399f1a2495412f3a54bd1b4e))
* **export:** serve EPUB as binary response ([0524fe3](https://github.com/Crimsab/oneday/commit/0524fe3020b360c55949fe6946475a85f0d4d9f1))
* **gateway:** bound live transcript snapshot ([d999cee](https://github.com/Crimsab/oneday/commit/d999cee17f3cc91a572776f43859dbcef8330de3))
* **http:** compress responses and cache static assets ([f347a25](https://github.com/Crimsab/oneday/commit/f347a25bdc624b54cb9fac3e45be70cc05cea12a))
* **rag:** persist norms across gateway processes ([e91cc95](https://github.com/Crimsab/oneday/commit/e91cc95f7b7efca14f6f189303feffddb73d4a0b))
* **rag:** select top results with bounded heap ([5036dcc](https://github.com/Crimsab/oneday/commit/5036dcc8e6195732b8689459b25393972a96efee))
* **search:** index history with FTS5 trigrams ([fa1b7c2](https://github.com/Crimsab/oneday/commit/fa1b7c25e92d4ab97f6afcdbf9cfaa1e8eef58fb))
* **sse:** slow external state reconciliation ([c856aab](https://github.com/Crimsab/oneday/commit/c856aaba3507af86e8f9b9230f8d57bd92918eaf))
* **telemetry:** batch diagnostic export queries ([fa5558d](https://github.com/Crimsab/oneday/commit/fa5558da189009d02d186ec656f825ef607d4feb))
* **timeline:** index message alternatives ([6938519](https://github.com/Crimsab/oneday/commit/693851973dfca9f3ce20925727a2ae0e20320c85))
* **timeline:** skip sealed snapshot recapture ([b7c3c39](https://github.com/Crimsab/oneday/commit/b7c3c3906582de99e317bed68096de9d07f45708))
* **timeline:** store row-level snapshot deltas ([0f83b0a](https://github.com/Crimsab/oneday/commit/0f83b0a6a7aa3882493ccf5a37d74025d2a11a2e))
* **visuals:** index known map relationships ([592688b](https://github.com/Crimsab/oneday/commit/592688be904387ecfcb79278621f67ce3e1c2c21))
* **web:** coalesce story reads and SSE setup ([603490b](https://github.com/Crimsab/oneday/commit/603490b8ffcafac1ee7f693680052f9c2d402385))
* **web:** isolate react runtime chunk ([5e2cc1a](https://github.com/Crimsab/oneday/commit/5e2cc1aef21badb2ce15e366d2163aca5185f331))
* **web:** isolate streaming transcript updates ([b464aa6](https://github.com/Crimsab/oneday/commit/b464aa6223620b5a7b4db12c6b2d07b8a2e9dc16))
* **web:** lazy load overlay workspace ([816942a](https://github.com/Crimsab/oneday/commit/816942a84fe13964e0eaf6b56a3ab9179309a878))
* **web:** split production vendor chunks ([538fffd](https://github.com/Crimsab/oneday/commit/538fffdef6c93b475b86e9717b3e1cb8dbf535a4))

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
