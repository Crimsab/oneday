# Summary 38.2 - Canonical map and rich export

`/map` is now a first-class browser module. Its accessible SVG renders only the
player-visible `locations` and `location_edges` projection, filters edges whose
endpoints are not known, and highlights the canonical current location.

Branch history exports Markdown, JSON, standards-compliant EPUB 3, and a
replay-v1 JSON manifest. EPUB stores `mimetype` uncompressed and includes
container, OPF, nav, and escaped XHTML. Replay export contains branch-scoped
messages, chapters, image/audio public URLs, and immutable lineage without file
paths. Desktop/mobile E2E covers map topology and EPUB/replay downloads.
