mermaid.min.js (Mermaid 11.16.1 UMD, dist/mermaid.min.js) is NOT committed.

Like the Relearn theme, it is fetched per-environment:
  - CI: the "Initialize theme and assets" step in .github/workflows/update-doc.yml
  - local: see hack/doc-site/hugo/README.md

Why vendored at all: the `railroad` shortcode (layouts/partials/custom-header.html)
needs Mermaid >= 11.16 — railroad diagrams are beta and newer than the theme's
bundled Mermaid. Serving it as a local static asset keeps the rendered site free of
runtime CDN calls. MIT License — https://github.com/mermaid-js/mermaid

Fetch it the way CI does — these bytes are published to the site and executed in
every visitor's browser, so they are checked rather than trusted:

  curl -fsSL --proto '=https' --tlsv1.2 -o mermaid.min.js \
    https://cdn.jsdelivr.net/npm/mermaid@11.16.1/dist/mermaid.min.js
  echo "18327bef70d96fb505fe7287d9f6a7362ebf07ff6576ddfaffb1a06f3e1a2954  mermaid.min.js" \
    | sha256sum -c -

The digest is the one pinned in update-doc.yml, and it is the file npm publishes:
`dist/mermaid.min.js` inside mermaid-11.16.1.tgz, whose own tarball hash matches the
integrity recorded in the registry. Moving the version means moving both.
