---
title: "Keyword reference"
weight: 20
description: "Per-keyword reference card: every keyword form, its value shape, and the contexts where it is legal."
---


This document catalogs the `keyword: value` forms recognised inside
annotation blocks. The keywords come in two flavours:

- **Inline keywords** — one line, `keyword: value` shape, with the
  value classified by a [value shape](#value-shapes) (number, integer,
  boolean, string, …).
- **Body keywords** — a header line followed by indented continuation
  lines. The body's interpretation depends on the keyword (a flat
  token list for `Consumes:`, a YAML map for `SecurityDefinitions:`,
  a per-line sub-language for `Parameters:` / `Responses:` on
  `swagger:route`).

The reader-friendly orientation is in [annotations.md]({{% relref "annotations" %}})
(which annotation accepts which keywords, with examples); this file
is the **per-keyword reference card**. Implementers wanting the
formal productions should read [grammar.md]({{% relref "grammar" %}}).

---

## Table of contents

- [Reading the tables](#reading-the-tables)
- [Value shapes](#value-shapes)
- [Annotation contexts](#annotation-contexts)
- [Summary table](#summary-table)
- [Numeric validations](#numeric-validations) — `maximum`, `minimum`, `multipleOf`
- [Length / array / object validations](#length--array--object-validations) — `maxLength`, `minLength`, `maxItems`, `minItems`, `maxProperties`, `minProperties`, `patternProperties`
- [Format validations](#format-validations) — `pattern`, `unique`, `collectionFormat`
- [Schema decorators](#schema-decorators) — `default`, `example`, `enum`, `required`, `readOnly`, `discriminator`, `deprecated`
- [Parameter location](#parameter-location) — `in`
- [Meta single-line keywords](#meta-single-line-keywords) — `schemes`, `version`, `host`, `basePath`, `license`, `contact`
- [Body keywords](#body-keywords) — `consumes`, `produces`, `security`, `securityDefinitions`, `responses`, `parameters`, `extensions`, `infoExtensions`, `tos`, `externalDocs`, `tags`

---

## Reading the tables

Each keyword entry carries:

- **Name** — canonical spelling. This is what `Property.Keyword.Name`
  compares equal to. Comparisons are case-insensitive on the
  canonical spelling and on every alias.
- **Aliases** — alternate spellings the lexer accepts. They map to
  the canonical name at lex time; consumers never see alias values.
- **Value shape** — the lexical category of the value. See
  [value shapes](#value-shapes) for what each one means and how it
  surfaces to consumers.
- **Contexts** — the family-level scopes where the keyword is legal.
  Using a keyword outside its legal contexts emits a
  `CodeContextInvalid` diagnostic and the keyword is dropped from
  the affected block.

## Value shapes

The grammar's lexer classifies every value into one of these
shapes. The shape determines which Walker callback fires for the
property and which field of `Property.Typed` carries the parsed
value.

| Shape | Typed payload | Example value forms |
|-------|---------------|---------------------|
| `number` | `float64` (with optional `<`/`<=`/`>`/`>=`/`=` prefix) | `5`, `1.5`, `<10`, `>=0`, `=42` |
| `integer` | `int64` | `5`, `100` |
| `boolean` | `bool` | `true`, `false`, `1`, `0` |
| `string` | raw `string` | `^[a-z]+$`, `date-time`, `multipart/form-data` |
| `comma-list` | raw `string`; split on `,` by `Property.AsList()` | `http, https`, `a,b,c` |
| `enum-option` | typed `string` (closed-vocab match) | `csv`, `pipes` for `collectionFormat:` |
| `raw-block` | accumulated body lines on `Property.Body` | multi-line YAML, indented token lists |
| `raw-value` | the verbatim post-colon text on `Property.Value` | `42`, `"orange"`, `[1, 2, 3]` |

When typing fails (e.g. `maximum: notanumber`) the lexer emits a
`CodeInvalidNumber` / `CodeInvalidInteger` / `CodeInvalidBoolean`
diagnostic and the property reaches the Walker with a zero-value
payload. Consumers gate on `Property.IsTyped()` to skip
malformed-typed values; the corresponding builder field stays
unwritten.

## Annotation contexts

The closed set of contexts a keyword can legally appear in. A keyword
table entry's `Contexts` field combines these:

| Context | Meaning |
|---------|---------|
| `param` | Parameter doc on a `swagger:parameters` struct field, or a `+ name:` chunk inside `swagger:route Parameters:` |
| `header` | Header field on a `swagger:response` struct |
| `schema` | Top-level model or struct field on a `swagger:model` |
| `items` | Items-level (array element) validation on either parameter or schema |
| `route` | Route-level metadata under `swagger:route` |
| `operation` | Inline operation metadata under `swagger:operation` |
| `meta` | Package-level metadata under `swagger:meta` |
| `response` | Response-level decorations |

---

## Summary table

The full keyword surface, in the order the keyword table declares
them. Detailed entries follow this table.

| Keyword | Aliases | Shape | Contexts |
|---------|---------|-------|----------|
| `maximum` | `max` | number | param, header, schema, items |
| `minimum` | `min` | number | param, header, schema, items |
| `multipleOf` | `multiple of`, `multiple-of` | number | param, header, schema, items |
| `maxLength` | `max length`, `max-length`, `maxLen`, `max len`, `max-len`, `maximum length`, `maximum-length`, `maximumLength`, `maximum len`, `maximum-len` | integer | param, header, schema, items |
| `minLength` | `min length`, `min-length`, `minLen`, `min len`, `min-len`, `minimum length`, `minimum-length`, `minimumLength`, `minimum len`, `minimum-len` | integer | param, header, schema, items |
| `pattern` | — | string | param, header, schema, items |
| `maxItems` | `max items`, `max-items`, `max.items`, `maximum items`, `maximum-items`, `maximumItems` | integer | param, header, schema, items |
| `minItems` | `min items`, `min-items`, `min.items`, `minimum items`, `minimum-items`, `minimumItems` | integer | param, header, schema, items |
| `unique` | — | boolean | param, header, schema, items |
| `collectionFormat` | `collection format`, `collection-format` | enum-option (`csv`, `ssv`, `tsv`, `pipes`, `multi`) | param, header, items |
| `maxProperties` | `max properties`, `max-properties`, `maximum properties`, `maximum-properties`, `maximumProperties` | integer | schema |
| `minProperties` | `min properties`, `min-properties`, `minimum properties`, `minimum-properties`, `minimumProperties` | integer | schema |
| `patternProperties` | `pattern properties`, `pattern-properties` | string (regex) | schema |
| `default` | — | raw-value | param, header, schema, items |
| `example` | — | raw-value | param, header, schema, items |
| `enum` | — | raw-value | param, header, schema, items |
| `required` | — | boolean | param, schema |
| `readOnly` | `read only`, `read-only` | boolean | schema |
| `discriminator` | — | boolean | schema |
| `deprecated` | — | boolean | operation, route, schema |
| `in` | — | enum-option (`query`, `path`, `header`, `body`, `formData`) | param |
| `schemes` | — | raw-block (token list) | meta, route, operation |
| `version` | — | string | meta |
| `host` | — | string | meta |
| `basePath` | `base path`, `base-path` | string | meta |
| `license` | — | string | meta |
| `contact` | `contact info`, `contact-info` | string | meta |
| `consumes` | — | raw-block (token list) | meta, route, operation |
| `produces` | — | raw-block (token list) | meta, route, operation |
| `security` | — | raw-block (security requirements) | meta, route, operation |
| `securityDefinitions` | `security definitions`, `security-definitions` | raw-block (YAML map) | meta |
| `responses` | — | raw-block (response sub-language) | route, operation |
| `parameters` | — | raw-block (parameter chunk sub-language) | route, operation |
| `extensions` | — | raw-block (YAML map of `x-*` entries) | meta, route, operation, schema, param, header |
| `infoExtensions` | `info extensions`, `info-extensions` | raw-block (YAML map of `x-*` entries) | meta |
| `tos` | `terms of service`, `terms-of-service`, `termsOfService` | raw-block (prose paragraph) | meta |
| `externalDocs` | `external docs`, `external-docs` | raw-block (YAML map) | meta, route, operation, schema, field |
| `tags` | — | raw-block (YAML) | meta, route, operation |

---

## Numeric validations

Apply to numeric schema types (`integer`, `number`). On a typed
schema with a non-numeric type, these keywords emit
`CodeShapeMismatch` and drop. On a typeless schema (no `type:`
declared upstream), they apply best-effort.

### `maximum`

Upper bound on a numeric value. Alias: `max`.

The value may carry a leading comparison operator that becomes the
exclusive/inclusive bound:

- `maximum: 10` — inclusive (≤ 10).
- `maximum: <10` — exclusive (< 10).
- `maximum: <=10` — inclusive (same as no prefix).
- `maximum: =10` — inclusive.

Maps to `schema.maximum` and `schema.exclusiveMaximum`.

```go
// Limit is the cap on items per page.
//
// maximum: 100
// minimum: 1
type Limit int
```

— from `fixtures/enhancements/...` (any numeric-validation fixture).

### `minimum`

Lower bound on a numeric value. Alias: `min`. Same operator-prefix
shape as `maximum`. Maps to `schema.minimum` and
`schema.exclusiveMinimum`.

### `multipleOf`

Divisibility constraint. The value must be a positive number.
Aliases: `multiple of`, `multiple-of`. Maps to `schema.multipleOf`.

```go
// AllowedStep enforces increments of 5.
//
// multipleOf: 5
type AllowedStep int
```

---

## Length / array / object validations

`maxLength` / `minLength` apply only to **string-typed** schemas;
`maxItems` / `minItems` apply only to **array-typed** schemas;
`maxProperties` / `minProperties` / `patternProperties` apply only to
**object-typed** schemas. Using the wrong pairing emits
`CodeShapeMismatch` and drops the keyword.

The object keywords are also **full-Schema-only**: there is no
SimpleSchema (non-body parameter, response header, or items chain) form
for them in OAS v2. Placing one on a SimpleSchema site emits
`CodeUnsupportedInSimpleSchema` and drops it. (Whether the offending
schema is detected via shape or via SimpleSchema mode, the keyword is
always dropped — never silently kept on a type it can't validate.)

### `maxLength`

Maximum string length. Many aliases for ergonomic spelling:
`max length`, `max-length`, `maxLen`, `max len`, `max-len`,
`maximum length`, `maximum-length`, `maximumLength`, `maximum len`,
`maximum-len`. Maps to `schema.maxLength`.

### `minLength`

Minimum string length. Same alias set as `maxLength` with `min` in
place of `max`. Maps to `schema.minLength`.

### `maxItems`

Maximum array length. Aliases: `max items`, `max-items`,
`max.items`, `maximum items`, `maximum-items`, `maximumItems`. Maps
to `schema.maxItems`.

### `minItems`

Minimum array length. Same alias shape as `maxItems` with `min` in
place of `max`. Maps to `schema.minItems`.

```go
// Tags is a non-empty, bounded list.
//
// minItems: 1
// maxItems: 20
// unique: true
type Tags []string
```

### `maxProperties`

Maximum number of properties on an **object**-typed schema. Aliases:
`max properties`, `max-properties`, `maximum properties`,
`maximum-properties`, `maximumProperties`. Maps to
`schema.maxProperties`. Schema-only — there is no SimpleSchema
(param/header/items) equivalent in OAS v2.

### `minProperties`

Minimum number of properties on an **object**-typed schema. Same
alias shape as `maxProperties` with `min` in place of `max`. Maps to
`schema.minProperties`.

### `patternProperties`

Constrains the **names** of properties on an **object**-typed schema by
regex. The argument is a single regex string; each `patternProperties`
line adds one entry to `schema.patternProperties`, mapping the regex to
an empty value schema (`{}` — "any value allowed for matching property
names"). Repeated lines accumulate distinct entries. Aliases:
`pattern properties`, `pattern-properties`.

Like the `pattern` keyword, the regex is RE2-hygiene-checked: a value
that does not compile under Go's RE2 engine raises a
`CodeInvalidAnnotation` warning but is **preserved** on the schema
(downstream tools may use JSON Schema's wider regex dialect) — it is
never dropped silently.

```go
// MyObjectType is a free-form object with property-count bounds and a
// property-name pattern.
//
// minProperties: 1
// maxProperties: 10
// patternProperties: ^x-
//
// swagger:model MyObjectType
type MyObjectType map[string]interface{}
```

---

## Format validations

### `pattern`

A regex constraint on a string value. The pattern is preserved
verbatim on `schema.pattern`. The grammar runs a best-effort RE2
compile (Go's regex engine) on the value; if it fails, a
`CodeInvalidAnnotation` diagnostic surfaces with the compile error.
The value still lands on the schema — downstream tools may use
JSON Schema's wider regex dialect.

```go
// Slug is a URL-friendly identifier.
//
// pattern: ^[a-z0-9-]+$
type Slug string
```

### `unique`

Marks an array-typed schema as set-valued (no duplicates). Boolean.
Maps to `schema.uniqueItems`.

### `collectionFormat`

How an array value is serialised on the wire. Closed-vocab:

- `csv` — comma-separated (default).
- `ssv` — space-separated.
- `tsv` — tab-separated.
- `pipes` — pipe-separated.
- `multi` — repeated `?key=val&key=val2` (query params only).

Aliases: `collection format`, `collection-format`. Maps to
`parameter.collectionFormat` / `items.collectionFormat`. Schema-level
contexts ignore this keyword (it's a SimpleSchema concept; schemas
serialise via `application/json`).

When the source value doesn't match the closed vocab, the raw value
is preserved verbatim on the parameter (matches the original
behaviour where `pipe` as a typo for `pipes` round-trips).

```go
// Tags is the form-data array of label tokens.
//
// in: query
// type: array
// collectionFormat: csv
// items.type: string
type TagsParam []string
```

---

## Schema decorators

### `default`

Default value for a schema or simple-schema field. Raw-value shape —
the post-colon text is captured verbatim and coerced against the
resolved schema type at write time (`ParseDefault` /
`CoerceValue`).

Multi-line bodies are accepted for complex literals:

```go
// Limits is the throughput envelope.
//
// default:
//   {
//     "rps": 100,
//     "burst": 200
//   }
type Limits struct { ... }
```

Single-line form for primitives:

```go
// Page is the page number.
//
// in: query
// type: integer
// default: 1
type PageParam int
```

### `example`

An example value for the schema, surfaced in tooling. Same raw-value
shape as `default`. Maps to `schema.example` (or `parameter.example`
for SimpleSchema parameters).

### `enum`

A closed set of allowed values. Three accepted surface forms:

- **Comma list**: `enum: red, green, blue` — split on `,` and
  trimmed.
- **JSON array**: `enum: ["red", "green", "blue"]` — parsed via
  YAML/JSON.
- **Multi-line list with `-` markers**:
  ```
  enum:
    - red
    - green
    - blue
  ```

Each element is coerced against the resolved schema type. Maps to
`schema.enum`.

For string-typed enums driven by Go `const` declarations the
`swagger:enum` annotation is the more idiomatic surface — it picks
up the constant names AND their godoc descriptions and produces an
`x-go-enum-desc` extension alongside the enum values. The
`enum:` keyword is the manual override.

By default the const→value mapping that `swagger:enum` derives is folded
into the field's `description` **and** duplicated in `x-go-enum-desc`. Set
the scanner option `SkipEnumDescriptions: true` to keep the authored prose
as the description; the mapping then rides `x-go-enum-desc` only. This is
independent of `SkipExtensions` (set both to suppress the mapping entirely).

When a struct field references a named primitive (`Status State` →
`type State string`), an `enum:` line in the referenced type's own doc
comment is parsed into that definition's enum values; the surrounding
prose becomes its title/description and the `enum:` line never leaks into
the text.

### `required`

Marks a field as required. Boolean.

- On a `swagger:model` struct field: adds the field's name to the
  enclosing schema's `required` array.
- On a `swagger:parameters` struct field: sets `parameter.required`.
- On a `swagger:response` header: not applicable; the keyword is
  silently dropped (response headers don't carry `required`).

### `readOnly`

Marks a schema property as read-only. Aliases: `read only`,
`read-only`. Maps to `schema.readOnly`.

Schema-only — emitting `readOnly:` inside a SimpleSchema context
(non-body parameter, response header) emits
`CodeUnsupportedInSimpleSchema` and drops the keyword.

### `discriminator`

Marks the property as the discriminator for an `allOf` polymorphic
schema. Boolean. Writes the property's name onto the enclosing
schema's `discriminator` field. Schema-only. The property should also be
`required` (a subtype cannot be selected from an absent value). Subtypes
that `allOf`-embed the base inherit the discriminator; the discriminator
value for each is its definition name — a custom-value annotation
(`swagger:discriminatorValue`) is not implemented. See the
[Polymorphic models]({{% relref "/tutorials/polymorphic-models" %}})
tutorial.

### `deprecated`

Marks the carrying entity as deprecated. Boolean. On operations
(`operation.deprecated`) and routes (`operation.deprecated` on the
synthesised op) it writes the native OpenAPI 2.0 `deprecated` field.
OpenAPI 2.0 has no native `deprecated` on the Schema object, so on a
**model or model field** it emits `x-deprecated: true` instead. A
godoc-style `Deprecated:` paragraph (the pkgsite convention) is an exact
synonym for `deprecated: true`, recognised in any context — and is the
idiomatic form on a Go doc comment, where a bare `deprecated: true` line
reads as a malformed deprecation notice to Go linters. Because it carries
semantic intent, `x-deprecated` survives even when `SkipExtensions` is set.

---

## Parameter location

### `in`

Where the parameter value comes from. Closed-vocab:

- `query` — query string parameter.
- `path` — path-parameter substitution.
- `header` — request header.
- `body` — request body (JSON, etc.).
- `formData` — form-data body field (note: `form` accepted as an
  alias inside `swagger:route Parameters:` chunks; the lexer
  normalises to `formData` at the canonical surface).

A non-matching value emits a context-invalid diagnostic; the
parameter loses its `in` and may end up incorrectly classified
downstream.

```go
// PageParams declares pagination query parameters.
//
// swagger:parameters listItems
type PageParams struct {
	// in: query
	// minimum: 0
	Offset int `json:"offset"`

	// in: query
	// minimum: 1
	// maximum: 100
	// default: 20
	Limit int `json:"limit"`
}
```

---

## Meta single-line keywords

Single-line keywords under `swagger:meta`. Values are taken as-is
from the post-colon string.

### `schemes`

Accepted URL schemes. Flexible list — all forms below produce the
same `["http", "https"]` output:

```
Schemes: http, https
Schemes:
  - http
  - https
Schemes: http
  - https
```

Maps to `spec.schemes`. See [sub-languages.md]({{% relref "sub-languages" %}})
§flex-lists for the unified rule.

### `version`

API version string. Maps to `info.version`.

### `host`

Default host. Defaults to `localhost` when empty. Maps to
`spec.host`.

### `basePath`

URL base path. Maps to `spec.basePath`. Aliases: `base path`,
`base-path`.

### `license`

License declaration. Two forms accepted:

```
License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
```

…where the trailing token starting with a URL scheme becomes
`license.url` and the prefix becomes `license.name`. A bare name
with no URL is accepted too.

### `contact`

Contact declaration. Author writes a `Name <email> URL` triple, in
any order. The grammar recognises:

- `Name <email@example.com>` — Go's `net/mail.ParseAddress` form.
- `Name <email@example.com> http://example.com` — same + trailing
  URL.
- Just a URL, no name.

Aliases: `contact info`, `contact-info`. Maps to `info.contact`.

---

## Body keywords

Body keywords have a header line ending in `:` and indented
continuation lines. The body's structure depends on the keyword.
See [sub-languages.md]({{% relref "sub-languages" %}}) for the full
sub-language specifications; this section covers the keyword
shape.

### `consumes` / `produces`

Media-type lists. Same flex-list rule as `schemes:` — comma
inline, multi-line bare, YAML `-` markers, or any combination.
Maps to `consumes` / `produces` on the surrounding scope (spec,
operation).

```
Consumes:
  - application/json
  - application/xml

Produces: application/json
```

### `security`

Security-requirements list. Each line is one requirement of shape
`schemeName: scope1, scope2`. An empty scope list (`schemeName:`)
means "no scopes required, but the scheme must be active."

```
Security:
  api_key:
  oauth2: read, write
```

Maps to `security` (array of single-key maps).

### `securityDefinitions`

YAML map declaring security schemes. The body is parsed as YAML
directly into the `spec.securityDefinitions` shape — see
[OAS v2 §5.2.16](https://swagger.io/specification/v2/#securityDefinitionsObject).

```
SecurityDefinitions:
  api_key:
    type: apiKey
    in: header
    name: X-API-Key
  oauth2:
    type: oauth2
    flow: implicit
    authorizationUrl: https://example.com/auth
    scopes:
      read: read access
      write: write access
```

Aliases: `security definitions`, `security-definitions`. Meta-only.

### `responses`

Per-route / per-operation response declarations. Each line is one
response in the form `<code>: <tokens>`. See
[sub-languages.md §responses]({{% relref "sub-languages#responses" %}}) for the
full per-line grammar.

```
Responses:
  200: body:User the requested user
  404: description: not found
  default: response:genericError
```

### `parameters`

Per-route / per-operation parameter declarations. Body is a sequence
of `+ name:` chunks (the `+` is the chunk-start sigil; `-` is
accepted as an alias). See
[sub-languages.md §parameters]({{% relref "sub-languages#parameters" %}}) for
the full per-chunk grammar.

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

### `extensions` / `infoExtensions`

Vendor extension declarations as a YAML map. Keys must start with
`x-` or `X-`; non-`x-*` keys emit `CodeInvalidAnnotation` and drop.

- `extensions:` lands the entries on the surrounding scope
  (`spec.extensions`, `operation.extensions`, `schema.extensions`,
  `parameter.extensions`, `header.extensions`, …) — including on
  parameters and response headers.
- `infoExtensions:` is meta-only; entries land on `info.extensions`.

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

Aliases: `info extensions`, `info-extensions` (for `infoExtensions`).

### `tos`

Terms-of-service prose paragraph. Multi-line body is joined with
`\n` after dropping whitespace-only lines. Aliases:
`terms of service`, `terms-of-service`, `termsOfService`. Maps to
`info.termsOfService`. Meta-only.

### `externalDocs`

External documentation pointer as a YAML map with `description` and
`url` keys. Aliases: `external docs`, `external-docs`.

Emitted on:

- **`swagger:meta`** → the top-level `externalDocs` object (and, nested under a
  `Tags:` entry, that tag's `externalDocs`);
- **`swagger:route` / `swagger:operation`** → the operation's `externalDocs`;
- **`swagger:model`** (and any full Schema, e.g. a body parameter's schema)
  → the schema's `externalDocs`;
- a **struct field** → the property's `externalDocs`. On a `$ref`'d field
  (whose property is a bare `$ref`) it is lifted onto the wrapping `allOf`
  compound, alongside the field's `description` and `x-*` siblings.

An empty block (no `description`/`url`) is skipped rather than emitting a
bare `externalDocs: {}`. It is a **full-Schema-only** keyword: on a
SimpleSchema site (a non-body parameter, response header, or items
chain) it is dropped with a `CodeUnsupportedInSimpleSchema` diagnostic.

```
ExternalDocs:
  description: Reference documentation
  url: https://example.com/docs
```

---

### `tags`

Top-level tag declarations. Behaviour depends on context:

- In **`swagger:meta`** the body is a YAML sequence of tag objects emitted into
  the spec's top-level `tags` — each with a `name`, an optional `description`,
  a nested `externalDocs`, and any `x-*` vendor extensions:

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

- In **`swagger:route` / `swagger:operation`** it is a plain string list,
  unioned and deduplicated with the annotation's header-line tags onto the
  operation's `tags`.

---

## Cross-keyword interactions

A handful of keyword interactions are worth flagging:

- **`default` + `example` + `enum`** on the same field: all three
  may co-occur. The values are coerced against the resolved schema
  type independently. If `enum` is declared and `default` is not a
  member of it, no diagnostic fires today — downstream JSON Schema
  validation catches it.
- **`type` + numeric validations + `format`** on a body parameter:
  the schema dispatcher's `checkShape` gates numeric / length
  validations against the resolved type. `format` is type-blind
  (any format string lands).
- **`required` on a `$ref'd` field**: writes to the enclosing
  schema's `required` array (the standard JSON-Schema-draft-4
  shape). If the field has sibling overrides, the `$ref` rewrites
  into an `allOf` compound — see
  [grammar.md]({{% relref "grammar" %}}) §refoverride.
