---
title: "swagger:route"
weight: 150
description: "Declares an HTTP route + operation in one annotation."
---


## What it does

Declares an HTTP route + operation in one annotation. The header line carries
the method, path, optional tags, and the operation ID; the comment body
carries the operation's metadata (consumes / produces / schemes / security /
parameters / responses / extensions).

This is the **terser of the two operation-declaration annotations**. Most
go-swagger projects use `swagger:route` for hand-written operations; see
{{% relref "swagger-operation" %}} for the YAML-body alternative.

## Where it goes

On a function or variable declaration whose doc comment carries the
annotation. The Go entity itself doesn't have to be a handler — the annotation
publishes a path/operation independent of the carrier.

A godoc-style identifier may precede the annotation on the same comment line
(`// ListPets swagger:route GET /pets pets users listPets`); that leading
identifier is recognised as a godoc convention and is not part of the
annotation surface.

## Syntax

```ebnf
RouteBlock    = ANN_ROUTE , OperationArgs
              , [ Title ] , [ Description ] , RouteBody ;

OperationArgs = HTTP_METHOD , URL_PATH , { IDENT_NAME } , IDENT_NAME ;
```

The trailing `IDENT_NAME` is the operation ID; the run before it is the tag
list. The header line shape authors rely on:

```
swagger:route <METHOD> <path> [tag1 tag2 …] <operationID>
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

All [body keywords]({{% relref "keywords#body-keywords" %}}) legal in route
context (`consumes`, `produces`, `schemes`, `security`, `parameters`,
`responses`, `extensions`, `externalDocs`) plus inline `deprecated:` and a body
`tags:` list (a string list, unioned and deduplicated with the header-line
tags). The `Parameters:` and `Responses:` sub-languages are documented in
{{% relref "sub-languages#parameters" %}} and
{{% relref "sub-languages#responses" %}}.

## Example

```go
// ListPets swagger:route GET /pets pets users listPets
//
// List pets filtered by some parameters.
//
//     Consumes:
//       - application/json
//
//     Produces:
//       - application/json
//
//     Schemes: http, https
//
//     Security:
//       api_key:
//       oauth: read, write
//
//     Parameters:
//       + name: limit
//         in: query
//         type: integer
//         minimum: 1
//         maximum: 100
//
//     Responses:
//       200: body:[]Pet the pet list
//       default: response:genericError
func ListPets() {}
```

**Full example.** `fixtures/enhancements/routes-full-petstore-shape/handlers.go`.

## Deprecated / legacy forms

In the `Parameters:` body the per-parameter chunk sigil `+ name:` (used in the
sample above) is the historic chunk-start; `- name:` is accepted as a
YAML-friendly alias and is preferred for YAML-correctness. See the chunk
grammar in {{% relref "sub-languages#parameters" %}}.
