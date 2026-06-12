---
title: Reference
weight: 3
description: |
  The annotation vocabulary, keyword reference and the formal grammar the
  scanner parses.
---

codescan parses a small annotation language layered on top of Go doc comments.
The reference material currently lives alongside the source in
[`docs/`](https://github.com/go-openapi/codescan/tree/master/docs):

- [Annotations](https://github.com/go-openapi/codescan/blob/master/docs/annotations.md)
  — the full annotation vocabulary (`swagger:meta`, `swagger:route`,
  `swagger:model`, `swagger:parameters`, `swagger:response`, …).
- [Keywords](https://github.com/go-openapi/codescan/blob/master/docs/keywords.md)
  — every keyword recognized inside annotation blocks and where it applies.
- [Grammar](https://github.com/go-openapi/codescan/blob/master/docs/grammar.md)
  — the formal grammar the parser implements.
- [Sub-languages](https://github.com/go-openapi/codescan/blob/master/docs/sub-languages.md)
  — the embedded YAML / simple-schema surfaces.

> **TODO (scaffold)**: these documents are good candidates to migrate into the
> doc-site as first-class pages (one section per file), so they render with
> navigation and search rather than linking out to GitHub.
