mermaid.min.js (Mermaid 11.16.0 UMD, dist/mermaid.min.js) is NOT committed.

Like the Relearn theme, it is fetched per-environment:
  - CI: the "Initialize theme and assets" step in .github/workflows/update-doc.yml
  - local: see hack/doc-site/hugo/README.md

Why vendored at all: the `railroad` shortcode (layouts/partials/custom-header.html)
needs Mermaid >= 11.16 — railroad diagrams are beta and newer than the theme's
bundled Mermaid. Serving it as a local static asset keeps the rendered site free of
runtime CDN calls. MIT License — https://github.com/mermaid-js/mermaid

Fetch it with:
  curl -sL -o mermaid.min.js https://cdn.jsdelivr.net/npm/mermaid@11.16.0/dist/mermaid.min.js
