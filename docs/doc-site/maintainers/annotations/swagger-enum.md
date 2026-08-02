---
title: "swagger:enum"
weight: 50
description: "Marks a named type as an enum and collects its const values."
---

## Usage

```goish
// swagger:enum [ IDENT_NAME ]
```

{{% notice style="note" %}}
Not to be confused with the [`enum:` keyword]({{% relref "/maintainers/keywords/schema-validations-and-decorators#enum" %}}),
which produces the same spec keyword from the opposite direction: it takes the
members you write literally, typed from the schema it sits on, whereas this
annotation collects them from a Go `const` block, typed from the declared Go
type. Side-by-side in the [enumerations tutorial]({{% relref "/tutorials/enumerations" %}}).
{{% /notice %}}

## What it does

Marks a named type over a string, integer, number or boolean as an enum
and collects the type's `const` declarations.

Values come from the Go type-checker, so any constant expression is
collected — `iota` (including the implicit specs, which carry neither a
type nor a value), computed members (`1 << 3`), references to earlier
members, negative values, every integer base, values above `MaxInt64` in
an unsigned enum, rune literals (as code points), `true` / `false`, and
both string forms. The emitted `type` / `format` come from the **declared
Go type**, never from the members, so an `int8` enum is
`{integer, int8}` and reordering the const block cannot change the type.
A type declared over another named type keeps that type's format
(`type Kind strfmt.UUID` stays `format: uuid`).

Two shapes do not work: an alias to a basic type cannot host an enum (the
type-checker erases the alias, leaving nothing to collect), and a `rune`
or `byte` enum emits integers, which is what those types are on the wire.
See [Enumerations]({{% relref "/tutorials/enumerations" %}}).

- **Without `swagger:model`** (the default): the values are applied
  **inline on each model field that references the type** — the property
  gets an `enum` array plus an `x-go-enum-desc` extension carrying the
  per-value godoc descriptions in `<value> <doc-text>` shape. The enum
  type itself is not a standalone definition.
- **With `swagger:model`**: the enum becomes a **first-class definition**
  carrying the `enum` array (+ `x-go-enum-desc`), and referencing fields
  point at it via `$ref` — the general `swagger:model ⇒ definition + $ref`
  rule applied to enums.

If `swagger:enum` names a type for which no matching `const` values are
found, the enum semantics are dropped and the type falls through to
ordinary type resolution (typically a plain `$ref`, no `enum` array).

## Where it goes

On a named type declaration. The type's `const` values are discovered via
Go's type-system traversal; they do not need to live in the same file.
The values surface only when a model reaches the enum type through a field.

## Grammar (EBNF)

```ebnf
EnumBlock = ANN_ENUM , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
```

The optional `IDENT_NAME` names the type whose `const` values to collect.
On a type declaration the name is redundant, so the bare `swagger:enum`
form is accepted and infers the name from the declared type:
`swagger:enum Priority` and a bare `swagger:enum` on `type Priority …`
are equivalent.

## Supported keywords

[Schema-context keywords]({{% relref "/maintainers/keywords/schema-validations-and-decorators#schema-decorators" %}}). The
`enum:` keyword can ALSO be used inline on the type doc to force a value
set; when present, it overrides the const-derived values and the
`x-go-enum-desc` is recomputed (or dropped) accordingly.

## Example

A named type marked `swagger:enum` with `const` values, referenced by a
model field, lands the values on that property (not on a standalone
definition) together with the `x-go-enum-desc` extension:

{{< example
    go="concepts/models/models.go" goregion="enum"
    json="concepts/models/testdata/enum.json" >}}

By default the const→value mapping is folded into the property's
`description` **and** duplicated in `x-go-enum-desc`. Set the scanner
option `SkipEnumDescriptions: true` to keep the authored prose as the
description; the mapping then rides `x-go-enum-desc` only. See
[Vendor extensions]({{% relref "vendor-extensions" %}}).

**Full example.** `fixtures/enhancements/enum-overrides/types.go`.
