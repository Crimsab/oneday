# Phase 37 verification

## Requirement evidence

| Requirement | Evidence |
| --- | --- |
| AUDIO-01 | Committed active-branch lookup and provisional rejection; segmentation/order tests |
| AUDIO-02 | Story mode/autoplay plus narrator/protagonist/NPC inherit/on/off browser controls |
| AUDIO-03 | Entity/identity/form assignments, locking, major-voice uniqueness and explicit override tests |
| AUDIO-04 | V35 constrained assets/jobs with branch, commit, message, provider, voice, language, hash, timing, status |
| AUDIO-05 | Full cache-key unit matrix, pronunciation revision invalidation, safe regular-file/path checks |
| AUDIO-06 | Cloud Speech and persistent Piper HTTP contract tests plus explicit disabled/unreachable status |
| AUDIO-07 | Native playback, polling, retry/cancel race, multilingual lexicon, audit cleanup, redacted manifest, desktop/mobile E2E |
| TRACE-03 | TTS generation run is causally parented to the narrator run and tested end to end |

## Release gates

- Go: 536 tests / 25 packages; `go vet ./...` clean.
- Rust gateway: 47 tests; Clippy clean with the pre-existing
  `too_many_arguments` image-generation signature allowance.
- Browser: 100 Vitest tests; TypeScript and Vite production build clean.
- Playwright: 10/10 across desktop Chromium and Pixel 7 viewport.
- React Scan: exercised during Phase 37.3; no render loop; instrumentation and
  dependency removed before commit.
- Data: copied-live-DB preflight and live schema V35; integrity `ok`; no foreign
  key violations; two stories and 15 visual assets preserved.
- Runtime: live API/UI on `oneday.homelab.local`; providers explicitly disabled;
  safe default TTS `off`; zero unintended audio rows; restart count zero.

Result: passed.
