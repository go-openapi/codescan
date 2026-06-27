---
title: "swagger:strfmt"
weight: 160
description: "Marks a named type as a custom string format."
---

## Usage

```goish
// swagger:strfmt FORMAT_NAME
```

## What it does

Marks a named type as a custom string format.

Wherever the type appears as a field, the emitted schema is
`{type: string, format: <name>}`. Useful for `UUID`, `Email`, `URL`-style
types that have a Go type but should serialise as a JSON string with a
known format.

A field typed by the marked type emits with the format; the underlying
type does NOT appear as a top-level model definition (strfmt-tagged types
are replaced by their format at every reference). A slice carries the
format onto its items: `{type: array, items: {type: string, format: …}}`.

## Where it goes

On a type declaration whose underlying form is a string-marshalable type
(typically implementing `encoding.TextMarshaler` / `encoding.TextUnmarshaler`).
`swagger:strfmt` may also sit on a struct **field** doc to override just
that field's published format.

## Grammar (EBNF)

```ebnf
StrfmtBlock = ANN_STRFMT , IDENT_NAME , [ Title ] , [ Description ] ;
```

The required `IDENT_NAME` is the format name (`uuid`, `email`, `mac`, …) —
the entire surface of the annotation.

## Supported keywords

None at the type level beyond `swagger:strfmt` itself; the format name is
the entire surface.

## Example

A named type marked `swagger:strfmt` (here a `MarshalText`/`UnmarshalText`
hardware address) emits as `{type: string, format: …}` wherever it is
referenced — a field typed `MAC` comes out as `{type: string, format: mac}`:

{{< example
    go="concepts/models/models.go" goregion="strfmt"
    json="concepts/models/testdata/strfmt.json" >}}

Adding `swagger:model` opts the type into a **first-class definition**
carrying the full `{type: string, format: …}` schema, with referencing
fields pointing at it via `$ref` — the general
`swagger:model ⇒ definition + $ref` rule. Without `swagger:model`, the
format inlines at every reference.

A field-level override targets one field's format — e.g.
`// swagger:strfmt int64` on a `uint64` field emits
`{type: string, format: int64}`, a precision-safe, JSON-conformant string
encoding (the conformant alternative to the Go-specific `{integer, format:
uint64}` codescan emits for unsized/large ints by default).

**Full example.** `fixtures/enhancements/text-marshal/types.go`.
