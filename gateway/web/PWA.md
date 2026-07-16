# OneDay PWA

OneDay is installable from Chromium on Windows and Linux. The installed app is
still a client of the canonical OneDay server: it does not create a second
offline story database and it does not queue story mutations while offline.

## Cache boundary

The generated service worker precaches only the compiled application shell,
fonts and local brand assets. It has no general runtime cache. These server
paths are explicitly `NetworkOnly` and are also excluded from navigation
fallback:

- `/api/*`, including JSON requests, mutations, audio and the SSE event stream
- `/generated/*`, including generated and uploaded story assets

When the browser is offline, a previously loaded build can open and explain
that the server is required. Story content and generated assets never come from
the PWA cache.

## Install and update behavior

The app captures Chromium's native install prompt and offers an explicit
install action. A waiting service worker never activates automatically. OneDay
shows an update notice and reloads only after the user chooses **Update now**,
which avoids interrupting an active narrative action.

## Verification

```bash
bun run test
bun run test:pwa
```

`test:pwa` builds the production bundle and runs Chromium against Vite Preview.
It checks the manifest and installability through CDP, verifies service-worker
registration and cache contents, then confirms that the app shell can open
offline while `/api` and `/generated` requests fail instead of returning stale
data.
