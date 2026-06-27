---
title: "swagger:type"
weight: 170
description: "Replaces a field's or named type's inferred Swagger type with an inlined type."
---


## What it does

Replaces a field's (or named type's) inferred Swagger type with an
**inlined** type. `swagger:type` is an inline directive — it never emits a
`$ref`; the chosen type is rendered directly in place (the default
`$ref`-for-named-types is the *no-annotation* behaviour).

## Where it goes

On a type declaration, a struct field doc, OR a `swagger:parameters` field
doc.

{{% notice style="note" %}}
**On a parameter field** the override collapses the field to a simple
parameter — useful when a struct- or defined-typed field would otherwise
come out typeless (invalid Swagger 2.0). The argument is restricted to a
**scalar** or a **`[]`-wrapped scalar** there: the `inline` and type-name
forms are rejected with a diagnostic, since a non-body parameter has no
schema to inline a type into. A compatible `swagger:strfmt` on the same
field still rides as a supplementary format.
{{% /notice %}}

## Syntax

```ebnf
TypeBlock = ANN_TYPE , TYPE_REF , [ Title ] , [ Description ] ;
```

The required `TYPE_REF` is one of:

- a **scalar type** — `string`, `integer`, `number`, `boolean`, `object`
  (or a Go-builtin spelling such as `int64`, `uint32`);
- **`[]T`** — an array whose items are the inlined `T` (recursive:
  `[][]int64`, `[]Custom`);
- **`inline`** — expand the field's own Go type in place, instead of the
  `$ref` a named type would otherwise produce;
- a **known type name** — inline that type's schema (again, no `$ref`).

An unknown name falls back to inlining the field's Go type, with a
`validate.unsupported-type` diagnostic.

## Supported keywords

None — the override type is the entire surface.

## Example

Type-level override:

```go
// ULID is a Crockford-base32 unique identifier rendered as a string.
//
// swagger:type string
type ULID [16]byte
```

Fields typed `ULID` emit as `{type: string}` regardless of the underlying
`[16]byte` shape.

Field-level override:

```go
type Document struct {
	// Body is an opaque payload published as a string blob.
	//
	// swagger:type string
	Body json.RawMessage `json:"body"`
}
```

**Interaction with `swagger:strfmt`.** `swagger:type` wins on the type
axis; a `swagger:strfmt` format on the same field is kept only when
**compatible** with the resolved type (a `string` accepts any format,
numeric types accept the numeric width formats), otherwise it is dropped
with a shape-mismatch diagnostic. `swagger:strfmt` alone is unchanged. See
[`swagger:strfmt`]({{% relref "swagger-strfmt" %}}).

**Interaction with `swagger:model`.** On a *type declaration* that also
carries `swagger:model`, the override shapes the type's **first-class
definition** (e.g. `swagger:type string` + `swagger:model` → a
`{type: string}` definition) and referencing fields `$ref` it. The
field-level inline form above is the behaviour *without* `swagger:model`.

**Full example.** `fixtures/enhancements/named-struct-tags-ref/types.go`.

## Deprecated / legacy forms

- The `array` argument is **deprecated** — use `inline`, or `[]T` for an
  explicit element type. It still works, with a `validate.deprecated`
  warning.
- `file` as an argument is rejected with a diagnostic — use
  [`swagger:file`]({{% relref "swagger-file" %}}).
