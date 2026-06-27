---
title: "swagger:additionalProperties"
weight: 10
description: "Sets a schema's additionalProperties policy for keys beyond the named properties."
---


## What it does

Sets a schema's `additionalProperties` — the policy for keys beyond the
named properties. On a struct it **complements** the named properties; on
a map type it **overrides** the element-derived value schema; on a type
that resolved to a bare `$ref` it **defines** a clean object. See the
[Maps & free-form objects]({{% relref "/tutorials/maps-and-free-form-objects" %}})
tutorial.

## Where it goes

On a type declaration (alongside `swagger:model`). A field-level
equivalent exists as the
[`additionalProperties:` keyword]({{% relref "/maintainers/keywords#additionalproperties" %}}).

## Syntax

```ebnf
AdditionalPropertiesAnnotation = ANN_ADDITIONAL_PROPERTIES , ( BOOL_VALUE | ValueType ) ;
ValueType                      = TYPE_REF | IDENT_NAME | "[]" , ValueType ;
```

The required token is one of:

- **`true`** — allow arbitrary extra keys (`additionalProperties: true`);
- **`false`** — forbid extra keys, closing the object
  (`additionalProperties: false`);
- a **value type** — a primitive / Go-builtin / `[]T`, or a **known type
  name** (which resolves to a `$ref`, and is registered for discovery).
  This reuses the {{% relref "swagger-type" %}} value grammar, except a
  type name becomes a `$ref` rather than an inline expansion.

## Supported keywords

None of its own. It composes with `maxProperties` / `minProperties` /
[`patternProperties`]({{% relref "swagger-patternproperties" %}}).

## Example

```go
// Settings is an open object: named properties plus typed extra values.
//
// swagger:model
// swagger:additionalProperties integer
type Settings struct {
	Name string `json:"name"`
}
```

**Precedence — lowest priority.** `additionalProperties` only rides on an
`object`. If a prior rule fixed a non-object type (a `swagger:type`
scalar, `swagger:strfmt`, a special type), the marker is dropped with a
`CodeShapeMismatch` diagnostic. It has no OAS-2 SimpleSchema form, so it
never applies on a non-body parameter or response header.

**Full example.** `fixtures/enhancements/additional-properties/api.go`.
