---
title: Documentation site
description: Preview and build the MK documentation site, the writing rules for pages in docs/, and how the site is published.
---

# Documentation site

The Markdown in `docs/` is the source of truth. The Astro Starlight project in
`docs-site/` reads it directly and never copies or generates files under `docs/`.

The published site is at `https://syntax-syllogism.com/mk/docs/`. The same pages
also read well on GitHub.

## Preview and build

Run the docs project from `docs-site/`:

```bash
cd docs-site
npm ci
npm run dev
```

The docs site needs Node `>=22.12.0`. Check your Node version first.

`npm run build` makes the production build and checks every internal link.
`npm run preview` serves the finished build. Output goes to `docs-site/dist/`,
which Git ignores. The build fails if a link is broken, or if a page is missing
its `title` or `description`.

`docs-site/astro.config.mjs` sets the base path, the explicit sidebar, and the
shared theme. `docs-site/src/content.config.ts` loads `../docs` and maps
`docs/README.md` to the site index. Keep both files small. The theme,
Starlight styling, and Markdown link rewriting come from
`@syntax-syllogism/docs-theme`. The shared favicon is committed at
`docs-site/public/ss-wordmark.svg`.

The Go build and tests don't read `docs-site/`, so the two can be checked
separately.

## Writing rules

- Every page needs `title` and `description` frontmatter.
- A new page goes in two places: the list in [`docs/README.md`](README.md), and
  the sidebar in `docs-site/astro.config.mjs`.
- Link between pages in `docs/` with relative `.md` links, anchors included.
  They work on GitHub, and the build rewrites them to site routes.
- Links to files outside `docs/`, such as `serve.go` or `index.html`, must be
  full public GitHub URLs (`https://github.com/Syntax-Syllogism/MK/blob/<ref>/...`).
  A relative path that leaves `docs/` fails the link check.
- Write for readers who want to finish a task. Use short sections, and say what
  to do. Don't include project history.
- Don't write a run of seven `<` characters anywhere in a page. MK treats any
  file containing one as a CONFLICT card if it is pointed at `docs/`. Describe
  conflict markers in words instead.

## Publishing

`.github/workflows/docs.yml` runs the shared docs-publish workflow. It runs only
on the `release` branch of the public `Syntax-Syllogism/MK` repository. Pushes
to the private repository skip the job. The workflow builds the site and
deploys only its `mk/docs/` folder to `Syntax-Syllogism.github.io`.

The release configuration's `docsLinkBase` lets the release tool keep the
GitHub links in `docs/` pointing at the release it ships. Build the docs site
before a release.
