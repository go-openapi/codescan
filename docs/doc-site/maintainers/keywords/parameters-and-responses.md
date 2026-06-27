---
title: "Parameters & responses"
weight: 10
description: "Keywords that decorate swagger:parameters fields and swagger:response headers — the reduced OAS 2.0 SimpleSchema surface, plus the parameter location and response-level examples."
---

These keywords decorate `swagger:parameters` fields and `swagger:response`
headers. Both sites ride the reduced OAS 2.0 **SimpleSchema** surface: the
validations constrain a primitive (or array-of-primitive) value, but the
full-Schema-only keywords (`maxProperties` / `minProperties` /
`patternProperties` / `additionalProperties` / `readOnly` / `discriminator` /
`externalDocs`) do **not** apply here. The location keyword `in` and the
universal `name` keyword are at home on this page; the validations are shared
with [Schema validations & decorators]({{% relref "schema-validations-and-decorators" %}}).

## Summary

| Keyword | Aliases | Shape | Contexts |
|---------|---------|-------|----------|
| `in` | — | string (closed-vocab) | param |
| `name` | — | string | param, header, schema, items |
| `collectionFormat` | `collection format`, `collection-format` | string (closed-vocab) | param, header, items |
| `examples` | — | YAML map (mime → payload) | response |
| `maximum` | `max` | number | param, header |
| `minimum` | `min` | number | param, header |
| `multipleOf` | `multiple of`, `multiple-of` | number | param, header |
| `maxLength` | `max length`, `maxLen`, … | integer | param, header |
| `minLength` | `min length`, `minLen`, … | integer | param, header |
| `maxItems` | `max items`, `maximumItems`, … | integer | param, header |
| `minItems` | `min items`, `minimumItems`, … | integer | param, header |
| `pattern` | — | string | param, header |
| `unique` | — | boolean | param, header |
| `default` | — | raw-value | param, header |
| `example` | — | raw-value | param, header |
| `enum` | — | raw-value | param, header |
| `required` | — | boolean | param |

The validation rows (`maximum` … `required`) are **visiting** here: they behave
exactly as on schemas — see
[Schema validations & decorators]({{% relref "schema-validations-and-decorators" %}}).
Two SimpleSchema restrictions apply on this page: `required` is **dropped on
response headers** (it sets `parameter.required` only on a body/non-body param),
and the object / structural keywords (`maxProperties`, `minProperties`,
`patternProperties`, `additionalProperties`, `readOnly`, `discriminator`,
`externalDocs`) are **not legal** here — placing one on a SimpleSchema site
drops it with `CodeUnsupportedInSimpleSchema`.

## Worked example(s)

A parameter set, every field carrying the SimpleSchema validation surface:

{{< example
    go="concepts/validations/validations.go" goregion="param"
    json="concepts/validations/testdata/param.json" >}}

A response with a validated header (note `in` is absent on header fields):

{{< example
    go="concepts/validations/validations.go" goregion="header"
    json="concepts/validations/testdata/header.json" >}}

## Parameter location

### `in`

Where the parameter value comes from. Closed-vocab:

- `query` — query string parameter.
- `path` — path-parameter substitution.
- `header` — request header.
- `body` — request body (JSON, etc.).
- `formData` — form-data body field (note: `form` is accepted as an alias
  inside `swagger:route Parameters:` chunks; the lexer normalises it to
  `formData` at the canonical surface).

A non-matching value emits a context-invalid diagnostic; the parameter loses its
`in` and may end up incorrectly classified downstream. The keyword is
parameter-only — it has no meaning on a response header (the header name is the
location).

## Field naming

### `name`

Sets the published name of **any** field it decorates, overriding the `json:`
tag / Go field name. It is the one canonical field-naming keyword and works at
every field site: a `swagger:model` property, an interface method, a
`swagger:parameters` field (the parameter name), and a `swagger:response` header
field (the `Headers` map key). Being structural, it is stripped from the
description rather than leaking into it.

Precedence, most-explicit-wins and **identical in every context**:

```text
name: keyword  >  swagger:name annotation  >  json: tag  >  Go field name
```

[`swagger:name`]({{% relref "/maintainers/annotations/swagger-name" %}}) is the
older annotation form — still honoured, and idiomatic on interface methods — but
`name:` is the universal keyword. Using `swagger:name` in a parameter or
response-header context (where `name:` is canonical) is inert and now emits a
`context-invalid` diagnostic pointing you at the keyword.

## Wire serialisation

### `collectionFormat`

How an array value is serialised on the wire. Closed-vocab:

- `csv` — comma-separated (default).
- `ssv` — space-separated.
- `tsv` — tab-separated.
- `pipes` — pipe-separated.
- `multi` — repeated `?key=val&key=val2` (query params only).

Aliases: `collection format`, `collection-format`. Maps to
`parameter.collectionFormat` / `items.collectionFormat`. This is a
SimpleSchema-only concept — schema-level contexts ignore it (schemas serialise
via `application/json`). When the source value doesn't match the closed vocab,
the raw value is **preserved verbatim** on the parameter (so `pipe` as a typo for
`pipes` round-trips).

## Response examples

### `examples`

Response-level examples on a `swagger:response` struct — a YAML map whose
first-level keys are **mime types** and whose values are the example payloads,
populating the OAS2 `Response.examples` field. This is the **plural**,
response-scoped keyword; contrast the **singular**, schema/param/header-scoped
[`example`]({{% relref "schema-validations-and-decorators#example" %}}) decorator.
The `swagger:operation` YAML body carries `examples` natively (it is unmarshalled
straight into the spec types); this keyword is the struct-`swagger:response`
counterpart.

```go
// swagger:response widgetResponse
//
// examples:
//
//	application/json:
//	  name: alice
//	  count: 3
//	application/xml: "<widget><name>alice</name></widget>"
type WidgetResponse struct {
	// in: body
	Body Widget `json:"body"`
}
```

See also [Spec metadata]({{% relref "spec-metadata" %}}) for the document-level
keywords that frame these operations.
