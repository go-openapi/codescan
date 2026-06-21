---
title: Descriptions beside a $ref
weight: 50
description: |
  Control how a field's description and extensions are rendered when its type
  resolves to a $ref — wrapped in an allOf, emitted as direct siblings
  (EmitRefSiblings), or dropped (SkipAllOfCompounding).
---

When a struct field's Go type resolves to a named model, the field becomes a
`$ref`. Strict JSON Schema draft 4 (the dialect OpenAPI 2.0 is built on) says a
`$ref` *replaces* its siblings — so a `description`, a validation, or an `x-*`
extension written on that field cannot simply sit next to the `$ref`.

codescan's default is to preserve those decorations by wrapping the reference in
an **`allOf` compound**, which is the draft-4-correct shape. Three options tune
this behaviour. The decorations split into two classes:

- **description & extensions** — *siblings-eligible*: modern tooling (OpenAPI
  3.1 / JSON Schema 2020-12, most Swagger-UI renderers) reads them directly
  beside a `$ref`.
- **validations & `externalDocs`** — *compound-only*: they have no valid
  bare-`$ref` form, so they can only ride an `allOf` compound.

{{< code file="shaping/refsiblings/refsiblings.go" lang="go" region="model" >}}

## The default — an `allOf` wrapper

With no options set, the field's description and extension are preserved by
wrapping the `$ref` as the single member of an `allOf`; the decorations ride the
outer schema:

{{< code file="shaping/refsiblings/testdata/default.json" lang="json" >}}

This is the always-correct shape and needs no configuration — see also
[Decorating a `$ref`]({{% relref "/tutorials/model-definitions" %}}) in the
Model definitions tutorial.

## Emit siblings directly — `EmitRefSiblings`

Set `Options.EmitRefSiblings` to render the description and extensions as
**direct siblings** of the `$ref`, with no `allOf` wrapper — the leaner shape
modern tools expect:

```go
codescan.Run(&codescan.Options{
    Packages:        []string{"./..."},
    ScanModels:      true,
    EmitRefSiblings: true,
})
```

{{< compare left="shaping/refsiblings/testdata/default.json" leftlabel="Default — allOf wrapper"
            right="shaping/refsiblings/testdata/siblings.json" rightlabel="EmitRefSiblings: true" >}}

{{% notice style="info" %}}
`EmitRefSiblings` only changes the cases where nothing else forces a compound.
When the field also carries a **validation** or **`externalDocs`** (which cannot
live beside a bare `$ref`), the `allOf` wrapper is still emitted and the
description / extensions ride its outer schema.
{{% /notice %}}

## Drop the compound entirely — `SkipAllOfCompounding`

Some downstream consumers — notably go-swagger's code generator — expect a field
that points at a model to be a **bare `$ref`** and do not handle the
`allOf`-compounded shape. Set `Options.SkipAllOfCompounding` to never emit an
`allOf` compound:

```go
codescan.Run(&codescan.Options{
    Packages:             []string{"./..."},
    ScanModels:           true,
    SkipAllOfCompounding: true,
})
```

No compound is produced, so validations and `externalDocs` are dropped, and the
description and extension go with them — leaving a bare `$ref`:

{{< code file="shaping/refsiblings/testdata/skip.json" lang="json" >}}

Every dropped decoration is reported through `Options.OnDiagnostic` (code
`validate.dropped-ref-sibling`), so the loss is never silent. Combine it with
`EmitRefSiblings` to keep the description and extensions as siblings while still
dropping the compound-only validations:

```go
codescan.Run(&codescan.Options{
    Packages:             []string{"./..."},
    ScanModels:           true,
    EmitRefSiblings:      true, // keep description / x-* as $ref siblings
    SkipAllOfCompounding: true, // drop validations / externalDocs, no allOf
})
```

{{% notice style="note" %}}
`required:` is never affected by any of these options. It is a property of the
*parent* object (it lands in the parent's `required` list), not a sibling of the
`$ref`, so it is always preserved.
{{% /notice %}}

## `DescWithRef` (deprecated)

`Options.DescWithRef` predates `EmitRefSiblings` and covers only the narrow
**description-only** case: a `$ref`'d field whose sole decoration is a
description. By default that description is dropped; `DescWithRef` preserves it
by wrapping the `$ref` in a single-arm `allOf`.

{{< compare left="shaping/descref/testdata/off.json" leftlabel="Default — description dropped"
            right="shaping/descref/testdata/on.json" rightlabel="DescWithRef: true" >}}

{{% notice style="warning" %}}
`DescWithRef` is **deprecated** — prefer `EmitRefSiblings`, which preserves both
descriptions **and** extensions (as direct siblings). `DescWithRef` keeps its
original behaviour for compatibility and is a no-op when `EmitRefSiblings` is
set.
{{% /notice %}}
