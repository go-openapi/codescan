---
title: Forcing a conformant format
weight: 10
description: |
  Override a Go-derived format (e.g. the vendor uint64/uint32 formats) with an
  official, JSON-conformant one using a field-level swagger:strfmt.
---

codescan derives a JSON-Schema `type` and `format` from each Go type. For the
unsized and large integer kinds it emits **Go-specific vendor formats** —
`uint64` → `{type: integer, format: uint64}`, `uint32` → `{integer, uint32}`,
and so on. These round-trip cleanly back to Go, but they are **not part of the
Swagger 2.0 format set**, and a `uint64` value can exceed what a JSON number
safely represents.

When you need conformant, precision-safe output, place a **field-level**
`swagger:strfmt` on the field to override just that property's format. Overriding
to `int64` publishes the value as a string-encoded `{type: string, format: int64}`:

{{< example go="shaping/formats/formats.go" goregion="formats" golabel="model"
            json="shaping/formats/testdata/measurement.json" jsonlabel="#/definitions/Measurement" >}}

`Raw` keeps the default `{integer, uint64}` vendor format; `Bounded` carries
`swagger:strfmt int64`, so it renders as a string-encoded `int64`. The override
is per-field — the underlying Go type is untouched everywhere else.

{{% notice style="info" %}}
`swagger:strfmt` also names a custom *string* format on a type declaration (e.g.
a `UUID` type → `{string, format: uuid}`); see
[Model definitions → swagger:strfmt]({{% relref "/tutorials/model-definitions" %}}).
The `swagger:type` annotation is the related tool when you want to override the
whole **type**, not just its format — see
[Type discovery]({{% relref "type-discovery" %}}) and the
[`swagger:type` reference]({{% relref "/maintainers/annotations/swagger-type" %}}).
{{% /notice %}}
