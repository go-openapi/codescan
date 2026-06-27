---
title: "Schema validations & decorators"
weight: 20
description: "Keywords that constrain and decorate a model schema or struct field — bounds, lengths, patterns, enums, defaults, and structural markers."
---

These keywords decorate a `swagger:model` schema or any struct field doc comment.
The validations constrain a value; the decorators carry defaults, examples, and
structural markers. Several also apply to parameters and response headers — there
they ride the reduced SimpleSchema surface
([Parameters & responses]({{% relref "parameters-and-responses" %}})).

## Summary

| Keyword | Aliases | Shape | Contexts |
|---------|---------|-------|----------|
| `maximum` | `max` | number | param, header, schema, items |
| `minimum` | `min` | number | param, header, schema, items |
| `multipleOf` | `multiple of`, `multiple-of` | number | param, header, schema, items |
| `maxLength` | `max length`, `maxLen`, … | integer | param, header, schema, items |
| `minLength` | `min length`, `minLen`, … | integer | param, header, schema, items |
| `maxItems` | `max items`, `maximumItems`, … | integer | param, header, schema, items |
| `minItems` | `min items`, `minimumItems`, … | integer | param, header, schema, items |
| `maxProperties` | `max properties`, … | integer | schema |
| `minProperties` | `min properties`, … | integer | schema |
| `pattern` | — | string | param, header, schema, items |
| `patternProperties` | `pattern properties`, `pattern-properties` | string (regex) | schema |
| `additionalProperties` | `additional properties`, `additional-properties` | `true`/`false`/type | schema |
| `unique` | — | boolean | param, header, schema, items |
| `default` | — | raw-value | param, header, schema, items |
| `example` | — | raw-value | param, header, schema, items |
| `enum` | — | raw-value | param, header, schema, items |
| `required` | — | boolean | param, schema |
| `readOnly` | `read only`, `read-only` | boolean | schema |
| `discriminator` | — | boolean | schema |
| `deprecated` | — | boolean | operation, route, schema |

The shared rows above (param/header/schema/items) are detailed here; on parameters
and headers they behave the same, with the OAS 2.0 SimpleSchema restrictions noted on
the [Parameters & responses]({{% relref "parameters-and-responses" %}}) page.
`collectionFormat` and `in`/`name` live there too.

## Worked examples

Every validation on a model's fields, side by side with the schema it produces:

{{< example
    go="concepts/validations/validations.go" goregion="field"
    json="concepts/validations/testdata/field.json" >}}

The object-validation keywords constrain the *map* of properties rather than named
fields:

{{< example
    go="concepts/validations/validations.go" goregion="object"
    json="concepts/validations/testdata/object.json" >}}

## Numeric validations

Apply to numeric schema types (`integer`, `number`). On a typed schema with a
non-numeric type they emit `CodeShapeMismatch` and drop; on a typeless schema they
apply best-effort.

### `maximum` / `minimum`

Upper / lower bound on a numeric value (aliases `max` / `min`). The value may carry a
leading comparison operator that sets the exclusive/inclusive bound:

- `maximum: 10` — inclusive (≤ 10);
- `maximum: <10` — exclusive (< 10);
- `maximum: <=10` / `maximum: =10` — inclusive.

Map to `schema.maximum`/`exclusiveMaximum` and `schema.minimum`/`exclusiveMinimum`.

### `multipleOf`

Divisibility constraint; the value must be a positive number. Aliases `multiple of`,
`multiple-of`. Maps to `schema.multipleOf`.

## Length, array & object validations

`maxLength` / `minLength` apply only to **string**-typed schemas; `maxItems` /
`minItems` only to **array**-typed; `maxProperties` / `minProperties` /
`patternProperties` only to **object**-typed. The wrong pairing emits
`CodeShapeMismatch` and drops. The object keywords are additionally
**full-Schema-only** — no SimpleSchema (non-body param, header, items) form exists in
OAS 2.0, so on such a site they drop with `CodeUnsupportedInSimpleSchema`.

### `maxLength` / `minLength`

String length bounds. Many ergonomic aliases (`max length`, `max-length`, `maxLen`,
`maximumLength`, …; `min` likewise). Map to `schema.maxLength` / `schema.minLength`.

### `maxItems` / `minItems`

Array length bounds (aliases `max items`, `maximumItems`, …). Map to
`schema.maxItems` / `schema.minItems`.

### `maxProperties` / `minProperties`

Property-count bounds on an **object** schema (aliases `max properties`, …). Map to
`schema.maxProperties` / `schema.minProperties`. Schema-only.

### `patternProperties`

Constrains the **names** of properties on an object schema by regex. The argument is
one regex string; each line adds an entry to `schema.patternProperties` mapping the
regex to an empty value schema (`{}` — any value allowed). Repeated lines accumulate.
Aliases `pattern properties`, `pattern-properties`. The regex is RE2-hygiene-checked:
one that doesn't compile raises `CodeInvalidAnnotation` but is **preserved**.

For **typed** value schemas (a regex → primitive or model `$ref`), use the
decl-level [`swagger:patternProperties`]({{% relref "/maintainers/annotations/swagger-patternproperties" %}})
marker. `patternProperties` is JSON-Schema, beyond the Swagger 2.0 subset — see
[Maps & free-form objects]({{% relref "/tutorials/maps-and-free-form-objects" %}}).

### `additionalProperties`

Policy for keys beyond the named properties on an object schema: `true` (allow any),
`false` (close the object), or a **value type** (primitive / `[]T`, or a model name
→ `$ref`). On a map field it overrides the Go element schema; on a `$ref`'d field the
value rides an `allOf` sibling so the reference is kept. Aliases `additional
properties`, `additional-properties`. Lowest-priority and object-only — dropped with
`CodeShapeMismatch` on a non-object. The decl-level
[`swagger:additionalProperties`]({{% relref "/maintainers/annotations/swagger-additionalproperties" %}})
marker does the same on a type.

## Format

### `pattern`

A regex constraint on a string value, preserved verbatim on `schema.pattern` —
including backslash escapes (`\d`, `\.`, `\n` reach the spec as literal two-character
sequences). The grammar runs a best-effort RE2 compile; a failure surfaces
`CodeInvalidAnnotation` but the value still lands (downstream tools may use a wider
regex dialect).

### `unique`

Marks an array-typed schema as set-valued (no duplicates). Boolean. Maps to
`schema.uniqueItems`.

## Schema decorators

### `default`

Default value for a schema or simple-schema field. Raw-value shape — the post-colon
text is captured verbatim and coerced against the resolved schema type at write time
(`ParseDefault` / `CoerceValue`). Single-line for primitives (`default: 1`),
multi-line bodies for complex literals:

```go
// default:
//   { "rps": 100, "burst": 200 }
```

### `example`

An example value for the schema, surfaced in tooling. Same raw-value shape as
`default`. Maps to `schema.example` (or `parameter.example` for SimpleSchema). This
is the **singular**, schema-scoped keyword; for the **plural** response-scoped
[`examples`]({{% relref "parameters-and-responses#examples" %}}) (a map keyed by mime
type) see Parameters & responses.

### `enum`

A closed set of allowed values. Accepted forms: comma list (`enum: red, green`),
bracketed comma list (`enum: [red, green]`), JSON array (`enum: ["red","green"]`), or
a multi-line `-` list. Each element is coerced against the resolved type; maps to
`schema.enum`.

For string enums driven by Go `const`s the
[`swagger:enum`]({{% relref "/maintainers/annotations/swagger-enum" %}}) annotation is
more idiomatic — it picks up the constant names + godoc and produces
`x-go-enum-desc`. The `enum:` keyword is the manual override. (Set
`SkipEnumDescriptions: true` to keep the const→value mapping on `x-go-enum-desc` only,
out of the description.)

### `required`

Marks a field as required. Boolean.

- On a `swagger:model` field: adds the field name to the schema's `required` array.
- On a `swagger:parameters` field: sets `parameter.required`.
- On a `swagger:response` header: not applicable — silently dropped.

### `readOnly`

Marks a schema property read-only. Aliases `read only`, `read-only`. Maps to
`schema.readOnly`. Schema-only — inside a SimpleSchema context it drops with
`CodeUnsupportedInSimpleSchema`.

### `discriminator`

Marks the property as the discriminator for an `allOf` polymorphic schema. Boolean;
writes the property name onto the schema's `discriminator`. Schema-only. The property
should also be `required`. Subtypes that `allOf`-embed the base inherit it; each
subtype's discriminator value is its definition name. See
[Polymorphic models]({{% relref "/tutorials/polymorphic-models" %}}).

### `deprecated`

Marks the carrying entity deprecated. Boolean. On operations/routes it writes the
native OAS 2.0 `deprecated`; OAS 2.0 has no Schema-object `deprecated`, so on a model
or field it emits `x-deprecated: true`. A godoc `Deprecated:` paragraph is an exact
synonym recognised in any context — and is idiomatic on Go doc comments. Because it
carries intent, `x-deprecated` survives even under `SkipExtensions`.
