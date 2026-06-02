---
title: "Sub-languages"
weight: 30
---

# Sub-languages

The annotation body grammar is not a single language — it's a
top-level keyword grammar that embeds several smaller languages
inside specific body keywords. Each embedded language has its own
shape rules.

This document catalogs the embedded languages and how they fit
together. For the per-keyword surface, see
[keywords.md](./keywords.md); for the formal grammar that hosts
them, see [grammar.md](./grammar.md).

---

## Table of contents

- [Prose classification (TITLE / DESC)](#prose-classification)
- [Flex-list (Property.AsList)](#flex-list)
- [Parameters body grammar](#parameters)
- [Responses body grammar](#responses)
- [YAML extensions surface](#yaml-extensions)
- [Security requirements](#security-requirements)
- [Contact / License inline forms](#contact-license)

---

## Prose classification

Comment lines that don't match any keyword head OR YAML fence OR
annotation marker are classified as **prose** — free-form text. The
lexer splits prose into two token kinds:

- **TITLE** — the first paragraph of prose, expected to fit on a
  short summary line.
- **DESC** — every prose paragraph after the title (or following a
  blank line within the first paragraph).

Three heuristics decide the title-vs-desc boundary, evaluated in
order. The first to fire wins:

1. **Blank-line split.** Any blank line inside the prose run ends
   the title paragraph and starts the description.
2. **Closing punctuation.** If the first prose line ends with
   Unicode punctuation (`.`, `?`, `!`, `…`, `:`, …), the title is
   just that one line; everything after becomes description.
3. **Markdown ATX heading.** If the first prose line matches
   markdown's `# Heading` shape, the `#` markers are stripped and
   the remaining text becomes the title.

When no heuristic fires, the entire prose run is title (the schema
builder later collapses to a description-only schema when
appropriate).

### `Package <name>` prefix strip

The `swagger:meta` annotation's title comes from the package doc
comment, which by Go convention starts with `Package <name>`. The
spec builder strips that prefix before publishing:

```go
// Package petstore Petstore API.
//
// Description of the petstore service.
//
// swagger:meta
package petstore
```

Produces `info.title = "Petstore API."` (the `Package petstore`
prefix stripped) and `info.description = "Description of the
petstore service."`

Only the capital-P `Package` form is recognised — author prose like
"package this carefully" is not chopped.

### Comment-marker noise stripping

Block-comment routes (`/* swagger:route … */`) typically carry
indented continuation lines:

```go
/* swagger:route POST /pets pets createPet

	Create a pet based on the parameters.

	Consumes:
		- application/json
*/
func CreatePet() {}
```

The lexer strips the leading whitespace (`\t`, `*`, `/`, `|`) per
line via [`trimContentPrefix`](./grammar.md#preprocess) before
classification.

### Markdown semantics that survive

- **Dash lists** in descriptions are preserved verbatim. A line
  starting with `- foo` lands in the description as
  `"- foo"` (not `"foo"`).
- **`---` lines** open a YAML fence — see
  [YAML extensions](#yaml-extensions) below.

---

## Flex-list

Body keywords that publish a flat list of tokens (`schemes:`,
`consumes:`, `produces:`) accept multiple surface forms uniformly.
The unified reader is `Property.AsList()`.

### Accepted forms

```
# Inline, comma-separated
Schemes: http, https

# Multi-line, indented bare lines
Schemes:
  http
  https

# Multi-line, YAML-style dash markers
Schemes:
  - http
  - https

# Inline value plus indented continuation
Schemes: http
  - https

# All combinations of the above
Consumes: application/json, application/xml
  - application/protobuf
```

All five forms produce the same `["http", "https"]` (or
`["application/json", "application/xml", "application/protobuf"]`)
output.

### Algorithm

For each input line — `Property.Value` first (if non-empty), then
each line of `Property.Body`:

1. Trim surrounding whitespace.
2. Drop a leading `- ` YAML marker if present.
3. Re-trim whitespace.
4. Comma-split.
5. Trim each token; drop empties.

Aggregate into a single slice in source order.

### What flex-list does NOT touch

- **Enum values** (`enum: ...`) — their elements may themselves be
  complex (JSON arrays, quoted strings with commas). `enum:` keeps
  its raw-value path; the value coercion layer handles array /
  comma-list / multi-line shapes per the schema type.
- **Parameters chunks** — the `+ name:` chunk grammar is not a
  simple token list; see [§parameters](#parameters).
- **YAML structural bodies** — `securityDefinitions:`,
  `extensions:`, `infoExtensions:` parse the body as YAML directly;
  their structure isn't a flat list. See
  [§yaml-extensions](#yaml-extensions).

---

## Parameters

The `Parameters:` body in `swagger:route` and `swagger:operation`
carries a sequence of parameter declarations separated by `+ name:`
chunks (the `+` is the chunk-start sigil; `-` is accepted as an
alias for forward compatibility with proper YAML).

### Chunk shape

```
Parameters:
  + name: id
    in: path
    type: integer
    description: the item identifier
    required: true
  + name: limit
    in: query
    type: integer
    minimum: 1
    maximum: 100
    default: 20
  + name: body
    in: body
    type: User
    required: true
```

### Per-chunk fields

The fields are classified into **head fields** (consumed by the
orchestrator to populate the `*spec.Parameter` shell) and
**validation fields** (lowered to grammar properties and dispatched
through the standard validation pipeline).

**Head fields:**

| Field | Lands on | Notes |
|-------|----------|-------|
| `name:` | `parameter.name` | Required. Identifies the parameter. |
| `in:` | `parameter.in` | One of `path` / `query` / `header` / `body` / `formData`. `form` accepted as an alias for `formData`. |
| `type:` | `parameter.type` (for SimpleSchema) or determines the body $ref | For non-body: one of `string` / `integer` / `number` / `boolean` / `array`. For body: a Go ident referring to a `swagger:model`-declared type, optionally with `[]` array prefixes (`[][]Pet`). `bool` accepted as an alias for `boolean`. |
| `format:` | `parameter.format` or `parameter.schema.format` | Free-form string. Applied after validation dispatch so it doesn't interfere with default/example coercion. |
| `description:` | `parameter.description` | Free-form prose. |
| `required:` | `parameter.required` | Boolean. |
| `allowempty:` / `allowemptyvalue:` | `parameter.allowEmptyValue` | Boolean. |

**Validation fields:** any other recognised
[keyword](./keywords.md) — `min`, `max`, `minLength`, `maxLength`,
`minItems`, `maxItems`, `pattern`, `unique`, `collectionFormat`,
`default`, `example`, `enum`. These are looked up via
`grammar.Lookup` (which accepts canonical names + aliases) and
dispatched through the standard handlers seam.

### Empty chunks and unknown keys

- A bare `+` (or `-`) sigil with no follow-up content emits a
  `CodeInvalidAnnotation` diagnostic and is dropped. The legacy
  parser silently emitted an empty `Parameter{}` object — current
  behaviour rejects it.
- Unknown keys (typos like `defualt:`) emit
  `CodeInvalidAnnotation` and drop. The legacy parser silently
  discarded them.

### Body parameters

When `in: body`, the orchestrator looks up `type:` as either:

- A primitive (`string`, `integer`, `number`, `boolean`, `array`,
  `object`) — emits a typed schema with the primitive on
  `parameter.schema.type`.
- A Go ident — emits a `$ref` to `#/definitions/<Ident>`. With `[]`
  prefixes, wraps the ref in nested array schemas.

Validation properties on a body chunk apply to the schema, gated by
the schema's resolved type via `checkShape`. A `min: 0` on a body
chunk with `type: Pet` (object) emits `CodeShapeMismatch` and drops;
a `min: 0` with `type: integer` lands on the schema's `minimum`.

### Validation on SimpleSchema (non-body) parameters

For `in:` other than `body`, validation properties apply directly to
the parameter (not to a sub-schema). Type-gating still applies:
`minLength` on `type: integer` emits a diagnostic and drops.

---

## Responses

The `Responses:` body in `swagger:route` carries one response
declaration per line. Each line has the shape:

```
<code>: <token>*
```

where `<code>` is `default` (case-insensitive) or a decimal HTTP
status code, and `<token>` is either a `tag:value` form or an
untagged token.

### Recognised tags

| Tag | Value shape | Lands on |
|-----|-------------|----------|
| `body:` | Go ident with optional `[]` prefixes (`body:[]Pet`) | A `$ref` to `#/definitions/<name>`, array-wrapped per `[]` count |
| `response:` | Go ident referring to a `swagger:response`-declared type | A `$ref` to `#/responses/<name>` |
| `description:` | Free-form prose (rest of line) | `response.description` |

### Untagged token rules

- The **first untagged token** defaults to a response ref. The
  orchestrator resolves it against the operation's `responses` map
  first, then falls back to `definitions` — if found in definitions
  (not responses), it's silently promoted to a body ref.
- **Subsequent untagged tokens** accumulate into the description.

### Examples

```
Responses:
  200: User the user as returned                  # untagged → response="User", desc="the user as returned"
  200: body:User the user                         # body ref + description
  200: response:userResponse the user             # named response ref
  201: body:Pet the created pet
  404: description: not found
  default: response:genericError
  default: body:[]ErrorList the error list        # array-wrapped body ref
```

### Diagnostics

- **Unknown tag** (`200: weird:value`) — emits
  `CodeInvalidAnnotation` and drops the line.
- **Duplicate body/response tags** on one line
  (`200: body:Pet response:errors`) — emits
  `CodeInvalidAnnotation`; the line drops.
- **Space-separated `body Foo`** (instead of `body:Foo`) — detected
  as a likely typo and dropped with diagnostic. The legacy parser
  silently treated it as `response="body"` (a dangling ref to a
  non-existent response).
- **Unresolvable response ref** — when a response name appears in
  neither `responses` nor `definitions`, the line drops with
  diagnostic. The legacy parser emitted a dangling `$ref`.

### Empty value lines

A line like `204:` with nothing after the colon produces a Response
with the code and an empty description. This is intentional — some
authors want a `204 No Content` with no body and no description.

---

## YAML extensions

Several body keywords parse their body as YAML directly:

- `extensions:` and `infoExtensions:` — a YAML map of `x-*` entries.
- `securityDefinitions:` — a YAML map matching OAS v2's
  `securityDefinitions` shape.
- `externalDocs:` — a YAML map with `description` and `url` keys.

### Extension typing

Extension values are NOT coerced to strings — they preserve their
YAML-typed form: `bool`, `float64`, `string`, `[]any`, or
`map[string]any` for nested structures.

```
Extensions:
  x-feature-flags:
    - alpha
    - beta
  x-rate-limit:
    requests: 100
    window: 60
  x-internal: true
  x-version: 0.5
```

Produces (extract):

```json
"x-feature-flags": ["alpha", "beta"],
"x-rate-limit": {"requests": 100, "window": 60},
"x-internal": true,
"x-version": 0.5
```

### `x-*` name gating

Keys that don't start with `x-` or `X-` emit a
`CodeInvalidAnnotation` diagnostic and drop. The build still
succeeds. Authors who relied on the legacy "hard error on non-x-*"
behaviour see a diagnostic + a clean spec missing the typo'd key.

```
Extensions:
  x-good: 1
  not-good: 2   # → diagnostic, dropped
```

### YAML body delimitation

The YAML extension bodies use **indentation** to delimit. A line
that returns to the indentation level of the keyword head — or
introduces a sibling keyword — terminates the body. The grammar
also recognises `---` fence pairs around the body (matching the
`swagger:operation` YAML shape) and absorbs them silently.

---

## Security requirements

The `security:` body (in `swagger:meta`, `swagger:route`, and
`swagger:operation`) carries OAuth-style security requirements
where each line is one requirement.

### Shape

Each line: `schemeName: scope1, scope2, …`

- `schemeName` matches a scheme declared in
  `securityDefinitions`.
- Scope list is comma-separated; trimmed; empties dropped.
- An empty scope list (`schemeName:`) means "this scheme is
  required, no scopes." Common for `apiKey` and `basic`.

### Example

```
Security:
  api_key:
  oauth2: read, write
  oauth2: admin
```

Produces:

```json
"security": [
  {"api_key": []},
  {"oauth2": ["read", "write"]},
  {"oauth2": ["admin"]}
]
```

Each requirement is a single-key map; the array is an OR
relationship (the request satisfies security if it matches ANY
entry).

---

## Contact / License

Inline single-line meta keywords with structured value
parsing.

### Contact

The `contact:` value carries up to three components: name, email,
URL. Recognised forms:

```
Contact: Name <email@example.com> https://example.com
Contact: Name <email@example.com>
Contact: https://example.com
Contact: <email@example.com>
```

The grammar splits the value on the first URL prefix it finds
(`https://`, `http://`, `ftps://`, `ftp://`, `wss://`, `ws://`),
then parses the prefix portion as `Name <email>` via Go's
`net/mail.ParseAddress`.

- A malformed `Name <email>` head (e.g., unbalanced angle
  brackets) surfaces as an error from `Block.Contact()`; the
  meta builder propagates it as a build failure.
- An empty contact line produces an empty `Contact` value (no
  error, no diagnostic — equivalent to omitting the keyword).

Aliases: `contact info`, `contact-info`.

### License

The `license:` value is split similarly:

```
License: Apache 2.0 https://www.apache.org/licenses/LICENSE-2.0
License: MIT
License: https://opensource.org/licenses/Custom
```

Same URL-prefix detection. Everything before the URL is the
license name; the URL (when present) is the license URL.
Either part may be empty.

License does NOT use `mail.ParseAddress` — the name is taken as
raw text up to the URL boundary.

---

## Sub-language interactions

Two interaction points worth flagging:

- **Block-comment continuation lines and the
  parameters/responses sub-languages.** A
  `/* swagger:route … */` block with `Parameters:` inside requires
  the chunk-start sigils (`+ ` / `- `) to be at the start of the
  trimmed line. Block-comment continuation noise (`\t`, `*`) is
  stripped first; if your editor inserts a `*` continuation marker,
  the lexer handles it transparently.
- **Flex-list and `description:`** on a parameter chunk.
  `description:` is a head field, not a list — it does NOT comma-split.
  Authors who write `description: foo, bar` get a single description
  `"foo, bar"`, not two descriptions. (This was a real ambiguity
  in older versions of go-swagger; the current grammar resolves
  it cleanly.)
