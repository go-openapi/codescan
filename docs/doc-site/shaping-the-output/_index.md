---
title: Shaping the output
weight: 30
description: |
  How-to guides for the knobs that change how the same Go source renders into
  the spec — grouped by what they shape: scope & discovery, names & $refs,
  titles & descriptions, field types & formats, and response bodies.
---

The same annotated Go can render into the spec in more than one shape, and a
handful of [`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options)
(plus a few field-level annotations) let you choose. The guides are grouped by
*what* they shape:

- **Scope & discovery** — which packages are read and which types become
  definitions.
- **Names & `$ref`s** — the names definitions are published under and how
  references render.
- **Titles & descriptions** — the human-readable text the spec carries.
- **Field types & formats** — how an individual property renders.
- **Response bodies** — describing a payload without a dedicated
  `swagger:response` struct.

{{< children type="card" description="true" >}}

Each guide is task-oriented — *"I want the output to look like this"* — and shows
the **same input rendered both ways**, as before/after golden output the example
tests verify. For the field-by-field meaning of every option, see the
[Options reference]({{% relref "options" %}}) or the
[`Options` godoc](https://pkg.go.dev/github.com/go-openapi/codescan#Options).
