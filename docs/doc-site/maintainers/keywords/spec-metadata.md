---
title: "Spec metadata"
weight: 50
description: "Top-of-document keywords authored under swagger:meta — version, host, base path, license, contact, terms of service — plus the cross-cutting vendor-extension and external-docs keywords."
---

These keywords author the spec's top-level fields. Most live in the package doc
comment carrying the `swagger:meta` block; a couple are cross-cutting and merely
have their *home* here — `extensions` lands on whatever scope it decorates, and
`externalDocs` rides meta, operations, schemas, and struct fields alike. The
remaining document-level concerns (`schemes`/`consumes`/`produces`, `security`/
`securityDefinitions`, the meta `tags` form) are owned by sibling pages and only
visit this one.

## Summary

| Keyword | Aliases | Shape | Home |
|---------|---------|-------|------|
| `version` | — | string | here |
| `host` | — | string | here |
| `basePath` | `base path`, `base-path` | string | here |
| `license` | — | `Name [URL]` | here |
| `contact` | `contact info`, `contact-info` | `Name <email> [URL]` | here |
| `tos` | `terms of service`, `terms-of-service`, `termsOfService` | prose | here |
| `infoExtensions` | `info extensions`, `info-extensions` | `x-*` YAML map | here |
| `extensions` | — | `x-*` YAML map | here (cross-cutting) |
| `externalDocs` | `external docs`, `external-docs` | `{description, url}` | here (cross-cutting) |
| `schemes` | — | flex-list | {{% relref "/maintainers/keywords/routes-and-operations" %}} |
| `consumes` / `produces` | — | flex-list | {{% relref "/maintainers/keywords/routes-and-operations" %}} |
| `security` | — | YAML | {{% relref "/maintainers/keywords/security" %}} |
| `securityDefinitions` | `security definitions`, `security-definitions` | YAML map | {{% relref "/maintainers/keywords/security" %}} |
| `tags` | — | YAML sequence | {{% relref "/maintainers/keywords/routes-and-operations#tags" %}} |

The visiting rows are documented where they primarily apply: `schemes`,
`consumes`, and `produces` are document-wide defaults overridden per operation
([Routes & operations]({{% relref "/maintainers/keywords/routes-and-operations" %}})); `security` and
`securityDefinitions` are detailed under [Security]({{% relref "/maintainers/keywords/security" %}});
the `swagger:meta` tag-objects form of `tags` is described under
[Routes & operations]({{% relref "/maintainers/keywords/routes-and-operations#tags" %}}).

## Worked example

A complete `swagger:meta` block, side by side with the document-level spec it
produces:

{{< example
    go="concepts/meta/doc.go" goregion="meta"
    json="concepts/meta/testdata/meta.json" >}}

## Meta single-line keywords

Single-line keywords under `swagger:meta`. The value is taken as-is from the
post-colon string.

### `version`

API version string. Maps to `info.version`.

### `host`

Default host for the API. Defaults to `localhost` when empty. Maps to
`spec.host`.

### `basePath`

URL base path applied to every route. Maps to `spec.basePath`. Aliases:
`base path`, `base-path`.

### `license`

License declaration, in two accepted forms:

```
License: Apache 2.0 https://www.apache.org/licenses/LICENSE-2.0.html
```

The trailing token starting with a URL scheme becomes `license.url`; the prefix
becomes `license.name`. A bare name with no URL is accepted too. Maps to
`info.license`.

### `contact`

Contact declaration. The author writes a `Name <email> URL` triple, in any
order; the grammar recognises:

- `Name <email@example.com>` — Go's `net/mail.ParseAddress` form;
- `Name <email@example.com> https://example.com` — same, plus a trailing URL;
- just a URL, with no name.

Aliases: `contact info`, `contact-info`. Maps to `info.contact`.

## Document-level keywords

### `tos`

Terms-of-service prose paragraph. The multi-line body is joined with `\n` after
dropping whitespace-only lines. Aliases: `terms of service`, `terms-of-service`,
`termsOfService`. Maps to `info.termsOfService`. Meta-only.

### `infoExtensions`

Vendor-extension declarations as a YAML map, landed on `info.extensions`. Keys
must start with `x-` or `X-`; a non-`x-*` key emits `CodeInvalidAnnotation` and
drops. Meta-only. Aliases: `info extensions`, `info-extensions`.

```
InfoExtensions:
  x-logo:
    url: https://example.com/logo.png
    altText: Example
```

For the same map on the surrounding scope rather than `info`, use
[`extensions`](#extensions).

### `extensions`

Vendor-extension declarations as a YAML map, landed on the **surrounding scope**
rather than on `info`: `spec.extensions`, `operation.extensions`,
`schema.extensions`, `parameter.extensions`, `header.extensions`, and so on —
including on parameters and response headers. Keys must start with `x-` or `X-`;
a non-`x-*` key emits `CodeInvalidAnnotation` and drops.

```
Extensions:
  x-internal-id: 42
  x-feature-flags:
    - alpha
    - beta
  x-nested:
    enabled: true
    rate: 0.5
```

This keyword is cross-cutting — it is documented here as its home, but applies
wherever a YAML body is parsed. For the meta-only `info.extensions` variant see
[`infoExtensions`](#infoextensions).

### `externalDocs`

External-documentation pointer as a YAML map with `description` and `url` keys.
Aliases: `external docs`, `external-docs`.

Emitted on:

- **`swagger:meta`** → the top-level `externalDocs` object (and, nested under a
  `Tags:` entry, that tag's `externalDocs`);
- **`swagger:route` / `swagger:operation`** → the operation's `externalDocs`;
- **`swagger:model`** (and any full Schema, e.g. a body parameter's schema) →
  the schema's `externalDocs`;
- a **struct field** → the property's `externalDocs`. On a `$ref`'d field (whose
  property is a bare `$ref`) it is lifted onto the wrapping `allOf` compound,
  alongside the field's `description` and `x-*` siblings.

An empty block (no `description`/`url`) is skipped rather than emitting a bare
`externalDocs: {}`. It is a **full-Schema-only** keyword: on a SimpleSchema site
(a non-body parameter, response header, or items chain) it drops with a
`CodeUnsupportedInSimpleSchema` diagnostic.

```
ExternalDocs:
  description: Reference documentation
  url: https://example.com/docs
```

Like [`extensions`](#extensions), this keyword is cross-cutting; it is
documented here as its home.
