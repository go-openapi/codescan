---
title: "Security"
weight: 40
description: "Keywords that wire authentication — the requirements that gate a spec, route, or operation, and the scheme catalogue declared once in meta."
---

Two keywords carry authentication into the spec. `security` lists the
**requirements** that gate the document, a route, or a single operation;
`securityDefinitions` is the **scheme catalogue** — declared once in
`swagger:meta` and referenced by name from every requirement. A requirement is
only meaningful when the scheme it names is defined, so the two are almost always
authored together (see [Spec metadata]({{% relref "spec-metadata" %}}) for the
rest of the `swagger:meta` surface and [Routes & operations]({{% relref "/maintainers/keywords/routes-and-operations" %}})
for where per-route requirements live).

## Summary

| Keyword | Aliases | Shape | Contexts |
|---------|---------|-------|----------|
| `security` | — | YAML sequence (raw-block) | meta, route, operation |
| `securityDefinitions` | `security definitions`, `security-definitions` | YAML map (raw-block) | meta |

## Worked example

The scheme catalogue and the document-wide default requirement, declared once in
the package `swagger:meta` block — the `schemes` golden captures both
`securityDefinitions` and the top-level `security`:

{{< example
    go="concepts/security/doc.go" goregion="meta"
    json="concepts/security/testdata/schemes.json" >}}

A route then overrides that default with its own `Security:` requirement — here
`oauth2` with the `read` and `write` scopes:

{{< example
    go="concepts/security/routes.go" goregion="routes"
    json="concepts/security/testdata/route.json" >}}

## Keyword details

### `security`

A YAML sequence of **requirement objects** parsed from the `Security:` body. The
semantics are OAS 2.0:

- multiple keys **within one sequence item** are **ANDed** — all of those schemes
  are required together (`{api_key, oauth2}` in one item);
- **separate items** are **ORed** — satisfying any one item grants access;
- a scheme's value is its **scope list**, a flow (`[read, write]`) or block list.
  For non-scoped schemes (`apiKey`, `basic`) the list is empty (`api_key: []`),
  meaning the scheme is required with no scopes;
- an empty top-level `Security: []` on an **operation** emits an explicit empty
  requirement — an intentional public opt-out that overrides the document-wide
  default rather than inheriting it.

A bare top-level mapping (`api_key:` / `oauth2: read, write`, comma-split scopes)
is still read as one OR requirement per key for back-compatibility. Maps to
`security` on the enclosing object. Legal in `swagger:meta` (the document
default), `swagger:route`, and `swagger:operation`. The full per-line body
grammar lives at
[sub-languages §security requirements]({{% relref "sub-languages#security-requirements" %}}).

### `securityDefinitions`

A YAML map, parsed directly into the `spec.securityDefinitions` shape — each entry
is a named scheme (`apiKey`, `oauth2`, `basic`) with its OAS 2.0 fields (`type`,
`in`, `name`, `flow`, `authorizationUrl`, `tokenUrl`, `scopes`, …); see
[OAS v2 §securityDefinitionsObject](https://swagger.io/specification/v2/#securityDefinitionsObject).
Aliases `security definitions`, `security-definitions`. **Meta-only** — the
scheme catalogue is declared once at the top of the document and referenced by
name from every `security` requirement. Its detail anchor is `#securitydefinitions`.
