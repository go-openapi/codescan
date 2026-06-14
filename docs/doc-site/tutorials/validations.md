---
title: Validations
weight: 30
description: |
  Drive JSON-Schema validations from field doc comments — numeric ranges,
  length and array bounds, patterns, formats, and enums — and understand the
  reduced surface on parameters and headers.
---

Validations are keyword-driven: you write `keyword: value` lines in a field's
doc comment and they become `minimum`, `maxLength`, `pattern`, `enum`, and the
rest of the validation surface on that property. Each pane below pairs the
annotated Go (left) with the fragment the scanner emits (right), from the
test-covered [`docs/examples/concepts/validations`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/validations)
package.

For the per-keyword reference card — value shapes, aliases, and legal contexts —
see [Keywords]({{% relref "/maintainers/keywords" %}}).

## On a model field — the full surface

A `swagger:model` field accepts the full JSON-schema validation vocabulary:

- **Numeric** — `minimum`, `maximum`, `multipleOf` (on `Price`).
- **Length** — `min length` / `max length` (on `Name`).
- **Arrays** — `min items` / `max items` / `unique` (on `Tags`).
- **Pattern** — a regular expression (on `SKU`).
- **Enum** — a fixed value set (on `Grade`).
- **Required** — `required: true` lifts the property into the schema's
  object-level `required` array (`sku`).

{{< example go="concepts/validations/validations.go" goregion="field"
            json="concepts/validations/testdata/field.json" jsonlabel="#/definitions/Product" >}}

## On parameters — the simple-schema surface

{{% notice style="info" %}}
**Simple schemas have a reduced surface.** Parameters other than `in: body`, and
response headers, are *simple schemas* in OpenAPI 2.0 — not full JSON schemas.
They accept the validation subset (`maximum`/`minimum`/`multipleOf`,
`maxLength`/`minLength`/`pattern`, `maxItems`/`minItems`/`uniqueItems`, `enum`,
plus the simple-schema-only `collectionFormat`) but **not** schema-only
constructs. A schema-only keyword such as `readOnly` placed on a query parameter
is simply not emitted — `spec.Parameter` has nowhere to carry it.

A Go **map** field has no simple-schema representation either: on a non-body
parameter or a response header it is skipped with a
`validate.unsupported-in-simple-schema` warning. Maps are only representable on
a body schema (as `object` + `additionalProperties`).
{{% /notice %}}

The same numeric and length keywords work on a query parameter; arrays add
`collectionFormat`:

{{< example go="concepts/validations/validations.go" goregion="param"
            json="concepts/validations/testdata/param.json" jsonlabel="parameters on searchProducts" >}}

## On response headers

A response header is also a simple schema, so it takes the same reduced
validation set (here `minimum` on an integer header). Note headers carry no
`required` flag.

{{< example go="concepts/validations/validations.go" goregion="header"
            json="concepts/validations/testdata/header.json" jsonlabel="responses[rateLimited]" >}}

## On an object — property count and name patterns

The object-validation keywords constrain a free-form object as a whole rather
than named fields: `minProperties` / `maxProperties` bound the property count,
and `patternProperties` permits properties whose name matches a regex. They are
schema-only — kept on an object-typed model, stripped (with a diagnostic) on a
scalar model or a simple-schema parameter.

{{< example go="concepts/validations/validations.go" goregion="object"
            json="concepts/validations/testdata/object.json" jsonlabel="#/definitions/Attributes" >}}

## What's next

- [Examples & defaults]({{% relref "/tutorials/examples-and-defaults" %}}) —
  attach example values and defaults.
- [Other type decorators]({{% relref "/tutorials/other-type-decorators" %}}) —
  `readOnly` and `deprecated`.
- [Keyword reference]({{% relref "/maintainers/keywords" %}}) — every keyword,
  its value shape, and where it is legal.
