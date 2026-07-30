# Documentation site

The OneDay documentation site is generated with Material for MkDocs and
published through GitHub Pages. Markdown in the repository remains the only
source of truth; there is no separate GitHub Wiki repository to synchronize.

## Source layout

- `README.md` becomes the site home page.
- `docs/` contains user, operator, architecture, and contributor guides.
- `CHANGELOG.md`, `CONTRIBUTING.md`, `SUPPORT.md`, `SECURITY.md`, and
  `CODE_OF_CONDUCT.md` are included under **Project**.
- `mkdocs.yml` owns navigation and the initial Material configuration.
- `scripts/prepare-docs-site.ts` stages only tracked or non-ignored
  documentation into `.mkdocs-site-src/`.

The staging step deliberately excludes databases, generated media, local
configuration, benchmark runs, build output, and every other ignored file.

## Build locally

Create a documentation-only virtual environment and install the pinned theme:

```bash
make docs-install
make docs-build
```

The strict build fails when configured navigation targets are missing, when
internal document links are broken, or when MkDocs emits configuration/build
warnings. Preview changes locally with:

```bash
make docs-serve
```

Then open `http://127.0.0.1:8000`.

## GitHub Pages deployment

`.github/workflows/docs.yml` builds and validates documentation changes on pull
requests and on `main`. Changes on `main` build a Pages artifact and deploy it
to the protected `github-pages` environment. The deployment job receives only
the permissions required by GitHub Pages: `pages: write` and `id-token: write`.

The public site is available at
[crimsab.github.io/oneday](https://crimsab.github.io/oneday/).

The generated Pages artifact is retained for one day. Python, Bun, MkDocs
caches, and the generated `site/` directory are not stored as long-lived
Actions artifacts.

## Adding or reorganizing a guide

1. Add the Markdown file below `docs/`.
2. Link it from `docs/README.md` and the root `README.md` when it is public.
3. Add it to `nav` in `mkdocs.yml`.
4. Run `make docs-build` and `bun scripts/check-docs.ts`.

The Material theme configuration lives in `mkdocs.yml`. OneDay-specific layout,
color, typography, focus, and responsive rules live in
`docs/stylesheets/extra.css`. Keep custom rules small and preserve Material's
accessible navigation and interaction behavior.
