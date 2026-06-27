---
title: "Routes & operations"
weight: 30
description: "Keywords carried in a swagger:route or swagger:operation block — the operation's transport metadata and its parameter and response bodies."
---

These keywords live inside a `swagger:route` or `swagger:operation` block. They
carry the operation's transport metadata — the URL schemes, the media types it
consumes and produces — and the two sub-language bodies that declare its
parameters and responses. Several of them double as document-wide defaults under
`swagger:meta` ([Spec metadata]({{% relref "spec-metadata" %}})), where an
operation-level value overrides the default.

## Summary

| Keyword | Aliases | Shape | Contexts |
|---------|---------|-------|----------|
| `schemes` | — | flex-list | meta, route, operation |
| `consumes` | — | flex-list | meta, route, operation |
| `produces` | — | flex-list | meta, route, operation |
| `responses` | — | sub-language (`<code>: <tokens>`) | route, operation |
| `parameters` | — | sub-language (`+ name:` chunks) | route, operation |
| `tags` | — | string list / tag objects | meta, route, operation |
| `deprecated` | — | boolean | operation, route, schema |
| `security` | — | YAML sequence (raw-block) | meta, route, operation |
| `externalDocs` | `external docs`, `external-docs` | `{description, url}` | meta, route, operation, schema |
| `extensions` | — | `x-*` YAML map | route, operation (cross-cutting) |

The visiting rows are documented where they primarily apply: `deprecated` is
detailed under
[Schema validations & decorators]({{% relref "schema-validations-and-decorators#deprecated" %}});
`security` (and its `securityDefinitions` catalogue) under
[Security]({{% relref "/maintainers/keywords/security" %}}); `externalDocs` and `extensions` under
[Spec metadata]({{% relref "spec-metadata" %}}).

## Worked examples

A `swagger:route` block — the path-line annotation plus its transport metadata
and a `Responses:` body — side by side with the path item it produces:

{{< example
    go="concepts/routes/routes.go" goregion="route"
    json="concepts/routes/testdata/route.json" >}}

The `swagger:operation` long form carries the same metadata in a YAML body,
including an inline `parameters` sequence:

{{< example
    go="concepts/routes/routes.go" goregion="operation"
    json="concepts/routes/testdata/operation.json" >}}

## Transport metadata

### `schemes`

Accepted URL schemes for the operation. Flexible list — comma inline, multi-line
bare, YAML `-` markers, or any combination all produce the same output
(`Schemes: http, https` ≡ a `- http` / `- https` block). See
[sub-languages §flex-lists]({{% relref "sub-languages" %}}) for the unified rule.

Maps to `schemes` on the enclosing operation. It is also a **document default**
under `swagger:meta` (`spec.schemes`), where an operation-level value overrides
the meta-level one — see [Spec metadata]({{% relref "spec-metadata" %}}).

### `consumes` / `produces`

Media-type lists — the request body MIME types the operation `consumes` and the
response MIME types it `produces`. Same flex-list rule as `schemes`: comma
inline, multi-line bare, YAML `-` markers, or any combination.

```
Consumes:
  - application/json
  - application/xml

Produces: application/json
```

Map to `consumes` / `produces` on the surrounding scope. Like `schemes`, both are
also `swagger:meta` document defaults overridden per operation.

## Body sub-languages

### `responses`

Per-route / per-operation response declarations. The body is one response per
line in the form `<code>: <tokens>`, where `<code>` is an HTTP status (or
`default`) and `<tokens>` names the body schema and/or description:

```
Responses:
  200: body:User the requested user
  404: description: not found
  default: response:genericError
```

The full per-line grammar lives at
[sub-languages §responses]({{% relref "sub-languages#responses" %}}).

### `parameters`

Per-route / per-operation parameter declarations. The body is a sequence of
`+ name:` chunks — the `+` is the chunk-start sigil (`-` is accepted as an
alias) — each chunk a small key/value block describing one parameter:

```
Parameters:
  + name: id
    in: path
    type: integer
    required: true
  + name: limit
    in: query
    type: integer
    default: 20
    minimum: 1
    maximum: 100
```

The full per-chunk grammar lives at
[sub-languages §parameters]({{% relref "sub-languages#parameters" %}}).

### `tags`

Tag declarations whose shape depends on context:

- In **`swagger:route` / `swagger:operation`** the body is a plain **string
  list**. It is unioned and deduplicated with the tags written on the
  annotation's header line, and the result lands on the operation's `tags`:

  ```
  Tags:
    - pets
    - store
  ```

- In **`swagger:meta`** the body is instead a YAML **sequence of tag objects**
  emitted into the spec's top-level `tags` — each with a `name`, an optional
  `description`, a nested `externalDocs`, and any `x-*` vendor extensions:

  ```
  Tags:
  - name: pets
    description: Everything about your Pets
    externalDocs:
      description: Find out more
      url: https://example.com/docs/pets
  - name: store
    x-display-name: Store
  ```

  The meta tag-objects form is also referenced from
  [Spec metadata]({{% relref "spec-metadata" %}}).

## Visiting keywords

These keywords also appear in a route/operation block but are detailed on their
home page:

- `deprecated` — marks the operation deprecated (native OAS 2.0 `deprecated`).
  See [Schema validations & decorators]({{% relref "schema-validations-and-decorators#deprecated" %}}).
- `security` — the per-route / per-operation requirement list (an empty
  `Security: []` on an operation is an explicit public opt-out).
  See [Security]({{% relref "/maintainers/keywords/security" %}}).
- `externalDocs` — the operation's external-documentation pointer.
  See [Spec metadata]({{% relref "spec-metadata" %}}).
- `extensions` — vendor `x-*` entries on the operation.
  See [Spec metadata]({{% relref "spec-metadata" %}}).
</content>
</invoke>
