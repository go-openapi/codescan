---
title: "swagger:operation"
weight: 110
description: "Declares an HTTP route + operation with a YAML-document body."
---

## Usage

```goish
// swagger:operation METHOD PATH [tag …] OPERATION_ID
```

## What it does

Declares an HTTP route + operation with a YAML-document body.

Same header line as {{% relref "swagger-route" %}} (method, path, optional
tags, operation ID), but with a different body shape: instead of the
structured `Parameters:` / `Responses:` keyword surface, `swagger:operation`'s
body is a single YAML document spelling out the OpenAPI operation object
directly.

Use `swagger:operation` when you want to author the operation in YAML (closer
to the OpenAPI spec text) or when the operation has shapes the keyword surface
doesn't cover.

## Where it goes

On a function or variable declaration whose doc comment carries the
annotation. The Go entity itself doesn't have to be a handler — the annotation
publishes a path/operation independent of the carrier.

## Grammar (EBNF)

```ebnf
InlineOperationBlock = ANN_OPERATION , OperationArgs
                     , [ Title ] , [ Description ] , InlineOperationBody ;

OperationArgs        = HTTP_METHOD , URL_PATH , { IDENT_NAME } , IDENT_NAME ;
```

`InlineOperationBody` is an `OPAQUE_YAML` document. The trailing `IDENT_NAME`
is the operation ID; the run before it is the tag list. The header line shape
authors rely on:

```
swagger:operation <METHOD> <path> [tag1 tag2 …] <operationID>
```

- `<METHOD>` — `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`, `OPTIONS`.
  Case insensitive.
- `<path>` — starts with `/`; supports path-parameter braces (`/items/{id}`).
  Only RFC 6570 Level-1 expansion (simple `{name}` substitution) is allowed;
  an inline regex constraint (`/items/{id:[0-9]+}`) is stripped to the bare
  form with a warning.
- `[tag1 tag2 …]` — optional whitespace-separated tag list (at least two
  characters each).
- `<operationID>` — the unique operation identifier.

## Supported keywords

None inside the YAML body — it is structurally YAML, not the keyword grammar.
The header line is the entire annotation surface.

## Example

{{< example
    go="concepts/routes/routes.go" goregion="operation"
    json="concepts/routes/testdata/operation.json" >}}

The `---` delimits the YAML body; everything between the fences is parsed as an
OpenAPI 2.0 operation object.

**Full example.** `fixtures/enhancements/parameters-map-postdecl/api.go`.

## Deprecated / legacy forms

`swagger:operation` also accepts a structured `Parameters:` body (shared with
{{% relref "swagger-route" %}}). In that body the per-parameter chunk sigil
`+ name:` is the historic chunk-start; `- name:` is accepted as a YAML-friendly
alias and is preferred for YAML-correctness. See the chunk grammar in
{{% relref "sub-languages#parameters" %}}.
