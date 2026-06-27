---
title: "swagger:response"
weight: 140
description: "Declares a Go struct as a named response object."
---

## Usage

```goish
// swagger:response [ IDENT_NAME ]
```

## What it does

Declares a Go struct as a named response object.

It is emitted into the spec's top-level `responses` map. Routes /
operations reference it by name via the response sub-language (`Responses:`
body in `swagger:route`, or the YAML `$ref` form in `swagger:operation`).

The struct's fields contribute the response shape:

- A field named `Body` (or carrying `in: body`) becomes the response
  body schema. The body may be a struct, a `$ref`'d model, **or a
  primitive** — `Body string` emits `schema: {type: string}`.
- Other fields default to response **headers**: a field with neither
  `Body`/`in: body` nor `in: header` is treated as a header, not a body
  property. A header's key comes from the `json:` tag / Go field name, or
  a `name:` keyword (e.g. `name: X-Rate-Limit`) — the canonical, preferred
  form, see the [`name` keyword]({{% relref "/maintainers/keywords/parameters-and-responses#name" %}}).
- An **anonymously embedded** struct marked `in: body` *is* the body (a
  `$ref` to the model), not a promotion of its fields.
- An `interface{}` / `any`-typed field emits an empty schema (`{}`, or
  `{type: array, items: {}}` for a slice) — "any type", valid OpenAPI 2.0.

## Where it goes

On a struct declaration.

## Grammar (EBNF)

```ebnf
ResponseAnnotation = ANN_RESPONSE , [ IDENT_NAME ] ;
```

The optional `IDENT_NAME` is the published response name (default: the
Go type's name). A `*` wildcard (`swagger:response *`) explicitly marks
the response as a **shared** one, registered at `#/responses/{name}` for
operations to `$ref` by name — see
[Sharing parameters & responses]({{% relref "/tutorials/sharing-parameters-and-responses" %}}).

The annotation opens a [`SchemaBlock`]({{% relref "grammar#schema-family" %}})
body.

## Supported keywords

- **Body field:** schema-context keywords.
- **Header field:** header-context keywords — numeric / length / format
  validations, `pattern`, `enum`, `default`, `example`,
  `collectionFormat`. `required:` is silently dropped (the OAS v2 Header
  object has no `required` field).

## Example

{{< example
    go="concepts/routes/routes.go" goregion="response"
    json="concepts/routes/testdata/response.json" >}}

Routes can then reference it via `response:genericError` in their
`Responses:` body.

**Full example.** `fixtures/enhancements/routes-full-petstore-shape/handlers.go`.
