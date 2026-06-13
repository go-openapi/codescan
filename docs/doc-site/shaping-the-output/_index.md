---
title: Shaping the output
weight: 30
description: |
  How-to guides for the knobs that change how the same Go source renders into
  the spec — $ref vs inline, alias handling, descriptions beside a $ref,
  nullable pointers, vendor extensions, and spec overlays.
---

The same annotated Go can render into the spec in more than one shape. A handful
of [`codescan.Options`](https://pkg.go.dev/github.com/go-openapi/codescan#Options)
let you choose: should an alias become a `$ref` or expand inline? Should a
field's description survive next to a `$ref`? Should pointer fields be marked
nullable?

Each guide here is task-oriented — *"I want the output to look like this"* — and
shows the **same input rendered both ways**, as before/after golden output the
example tests verify.

{{< children type="card" description="true" >}}

For the field-by-field meaning of each option, see the
[`Options` godoc](https://pkg.go.dev/github.com/go-openapi/codescan#Options).
