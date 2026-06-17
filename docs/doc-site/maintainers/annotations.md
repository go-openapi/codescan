---
title: "Annotations"
weight: 10
description: "The swagger:* annotation vocabulary: what each produces, where it attaches, and the keywords it admits."
---


Annotations are the `swagger:<name>` markers the scanner recognises in
Go doc comments. Each annotation classifies the surrounding
declaration — telling the scanner "this is a model definition", "this
is a route handler", "this is meta-information about the API" — and
opens the door for [keywords]({{% relref "keywords" %}}) inside the same comment
block.

There are twelve annotations. They divide cleanly by what they
attach to:

- **Spec-level**: `swagger:meta`.
- **Model declarations**: `swagger:model`, `swagger:strfmt`,
  `swagger:enum`, `swagger:allOf`, `swagger:alias`,
  `swagger:additionalProperties`, `swagger:patternProperties`.
- **Operation declarations**: `swagger:route`, `swagger:operation`.
- **Companion declarations**: `swagger:parameters`, `swagger:response`.
- **Local hints**: `swagger:ignore`, `swagger:name`, `swagger:type`,
  `swagger:file`, `swagger:default`.

This file is the **author-first reference**. Each entry covers:

- What the annotation does and what it produces in the spec.
- Where in the Go source it goes (package doc, type doc, field doc).
- The shape of any argument the annotation accepts.
- A short Go sample.
- A pointer to the keywords that are legal inside the block.
- A pointer to a real fixture in this repo for the full executable
  example.

For the per-keyword reference, see [keywords.md]({{% relref "keywords" %}}).
For the embedded sub-languages (`Parameters:` and `Responses:` body
grammars, YAML extensions, etc.), see
[sub-languages.md]({{% relref "sub-languages" %}}). For the formal grammar,
see [grammar.md]({{% relref "grammar" %}}).

---

## Table of contents

- [How annotations attach](#how-annotations-attach)
- [Annotation argument shapes](#annotation-argument-shapes)
- [`swagger:meta`](#swaggermeta)
- [`swagger:model`](#swaggermodel)
- [`swagger:strfmt`](#swaggerstrfmt)
- [`swagger:enum`](#swaggerenum)
- [`swagger:allOf`](#swaggerallof)
- [`swagger:alias`](#swaggeralias)
- [`swagger:route`](#swaggerroute)
- [`swagger:operation`](#swaggeroperation)
- [`swagger:parameters`](#swaggerparameters)
- [`swagger:response`](#swaggerresponse)
- [`swagger:ignore`](#swaggerignore)
- [`swagger:name`](#swaggername)
- [`swagger:type`](#swaggertype)
- [`swagger:additionalProperties`](#swaggeradditionalproperties)
- [`swagger:patternProperties`](#swaggerpatternproperties)
- [`swagger:file`](#swaggerfile)
- [`swagger:default`](#swaggerdefault)

---

## How annotations attach

An annotation is recognised when it appears at the start of a comment
line in a doc comment. Leading whitespace, the `//` marker, and any
`/* */` block-comment continuation noise are stripped — the lexer
applies the same content-prefix-trim that every other godoc-aware
tool does.

Annotations attach to whichever Go declaration owns the comment
group:

- **Package doc** (`// Package foo …` followed by `package foo`) —
  carries `swagger:meta`.
- **Type declaration** (`type T struct { … }`, `type T int`,
  `type T = Other`) — carries `swagger:model`, `swagger:strfmt`,
  `swagger:enum`, `swagger:allOf`, `swagger:alias`, `swagger:ignore`,
  `swagger:type`. Inside a grouped declaration (`type ( A …; B … )`)
  the comment on each individual spec is honoured independently — the
  annotation attaches to its own `TypeSpec`, not to the enclosing group —
  so two types in one group can carry distinct docs and annotations.
- **Function or variable declaration** (`func ServeAPI() { … }`,
  `var DoIt = func() { … }`) — carries `swagger:route`,
  `swagger:operation`. These two are recognised whether the annotation
  sits in the function's doc comment or **inside the function body**.
  A `swagger:model` or `swagger:parameters` declared on a type **local
  to a function body** is likewise discovered.
- **Struct field doc** — carries `swagger:name`, `swagger:type`,
  `swagger:ignore`, plus any of the [keyword reference]({{% relref "keywords" %}})
  entries legal in `schema` / `param` / `header` context.

One comment group may carry MORE than one annotation when the
combinations are semantically compatible — e.g. `swagger:model` +
`swagger:type` together overrides the auto-detected Go type while
still publishing the model. The grammar parses both and the builder
honours both.

The **first** annotation in source order wins as the "primary"
classifier — for example, a comment carrying `swagger:model` followed
by `swagger:ignore` produces a model (the ignore is silently
overridden because only the source-order-first annotation drives the
short-circuit). Subsequent annotations are still parsed and visible
via `Block.AnnotationKind()`-iteration, but the primary classifier
determines which builder owns the decl.

{{% notice style="warning" %}}
Recognition is purely positional: **any** comment line that begins with a
`swagger:<name>` token is treated as that annotation — even when you meant it as
prose. A description line like `swagger:type controls the emitted type` on a
type's doc comment is parsed as a `swagger:type` annotation. Keep annotation
names mid-sentence in descriptions (`The swagger:type directive …`) or wrap them
in backticks so the line does not *start* with the token.
{{% /notice %}}

## Annotation argument shapes

After the `swagger:<name>` head, an annotation may carry positional
arguments. The shapes:

- **No args**: `swagger:meta`, `swagger:ignore`, `swagger:enum`,
  `swagger:allOf`, `swagger:file`, `swagger:default` — bare
  annotation, the surrounding decl supplies the entity name.
- **One IDENT arg**: `swagger:model Pet`, `swagger:response
  errorResponse`, `swagger:strfmt uuid`, `swagger:name fullName`,
  `swagger:type integer`, `swagger:alias TimestampAlias` — the
  argument overrides or names the entity.
- **One IDENT arg, optional**: `swagger:model` (bare — derives the
  name from the Go decl) vs `swagger:model Pet` (overrides).
- **List of IDENT args**: `swagger:parameters listItems createItem`
  — declares the parameters group as legal for multiple operations.
- **Header line**: `swagger:route GET /pets pets users listPets` and
  `swagger:operation GET /pets users listPets` — a structured header
  carrying method, path, tags, and operation ID. See the
  per-annotation entries for the exact rules.

---

## `swagger:meta`

**What it does.** Declares the package as the OpenAPI spec
container. The scanner reads the package doc comment for top-level
spec fields: title (via [stripPackagePrefix]({{% relref "grammar#prose" %}}) of
the doc's first line), description, license, contact, host,
basePath, version, schemes, consumes, produces, securityDefinitions,
extensions, and the rest of the meta keyword surface.

**Where it goes.** On the package doc comment.

**Argument shape.** No args. Bare annotation.

**Sample.**

```go
// Package petstore Petstore API.
//
// The purpose of this application is to provide an application
// that is using plain Go code to define an API.
//
//     Schemes: http, https
//     Host: petstore.swagger.io
//     BasePath: /v2
//     Version: 1.0.0
//
//     Consumes:
//       - application/json
//
//     Produces:
//       - application/json
//
// swagger:meta
package petstore
```

**Legal keywords.** All [meta single-line keywords]({{% relref "keywords#meta-single-line-keywords" %}})
(`schemes`, `version`, `host`, `basePath`, `license`, `contact`) plus
the meta-scope [body keywords]({{% relref "keywords#body-keywords" %}})
(`consumes`, `produces`, `security`, `securityDefinitions`,
`extensions`, `infoExtensions`, `tos`, `externalDocs`, `tags`). A
`Tags:` block declares the spec's top-level `tags` (name, description,
nested `externalDocs`, `x-*` extensions per tag).

**Full example.** `fixtures/goparsing/spec/api.go`.

---

## `swagger:model`

**What it does.** Declares a Go type as a published model. The
scanner walks the type, emits a schema into the spec's `definitions`
map, and resolves cross-references between models.

**Where it goes.** On a type declaration (`type T struct { … }`,
`type T int`, `type T = Other`, …).

**Argument shape.** Optional IDENT — the name the model takes in
`definitions`. Default: the Go type's name. The name must be a plain
identifier (a JSON label), not a Go-qualified name — a dotted name such
as `utils.Error` is rejected with a warning and dropped. Cross-package
types are resolved automatically, so reference a model by its bare name.

**Ordering.** The descriptive prose must come **before** the
`swagger:model` line. The title/description split follows a heuristic:
a single-line comment **ending in a period** becomes the `title`; a
single-line comment **without** a trailing period becomes the
`description`; a **multi-line** comment uses the first line as `title`
and the remaining paragraphs as `description`. An annotation-first block
(the `swagger:model` line ahead of the prose) still publishes the model
but drops its title and description.

**Sample.**

```go
// Pet is the petstore's primary entity.
//
// swagger:model
type Pet struct {
	// ID is the unique identifier.
	ID int64 `json:"id"`

	// Name is the pet's display name.
	Name string `json:"name"`

	// Tags categorise the pet.
	Tags []string `json:"tags,omitempty"`
}
```

With a name override:

```go
// swagger:model PetWithExtras
type DetailedPet struct { … }
```

The type is published as `#/definitions/PetWithExtras`.

**Multiple names on one line.** A field group declaring several names
(`R, G, B, A uint8`) emits **one property per name**. A `json:` tag on
such a group cannot rename the individual fields — each keeps its own
name — though tag options still apply.

**Legal keywords.** All [schema]({{% relref "keywords#schema-decorators" %}})
keywords plus the
[length / array / numeric validations]({{% relref "keywords#numeric-validations" %}})
on field doc comments.

**Full example.** `fixtures/enhancements/named-struct-tags-ref/types.go`.

---

## `swagger:strfmt`

**What it does.** Marks a named type as a custom string format.
Wherever the type appears as a field, the emitted schema is
`{type: string, format: <name>}`. Useful for `UUID`, `Email`,
`URL`-style types that have a Go type but should serialise as a
JSON string with a known format.

**Where it goes.** On a type declaration whose underlying form is a
string-marshalable type (typically implementing `encoding.TextMarshaler`
or `encoding.TextUnmarshaler`).

**Argument shape.** Required IDENT — the format name (`uuid`, `email`,
`mac`, etc.).

**Sample.**

```go
// MAC is a hardware address rendered as a colon-separated hex string.
//
// swagger:strfmt mac
type MAC string

func (m MAC) MarshalText() ([]byte, error) { return []byte(m), nil }
func (m *MAC) UnmarshalText(b []byte) error { *m = MAC(b); return nil }
```

A field typed `MAC` emits as `{type: string, format: mac}`. The
underlying `MAC` type does NOT appear as a top-level model definition
(strfmt-tagged structs are replaced by their format at every
reference). A slice of the type (`[]MAC`) carries the format onto its
items: `{type: array, items: {type: string, format: mac}}`.

**With `swagger:model`.** Adding `swagger:model` to the strfmt type opts
it into a **first-class definition** carrying the full
`{type: string, format: …}` schema, and referencing fields point at it
via `$ref` — the general `swagger:model ⇒ definition + $ref` rule. Without
`swagger:model`, the format inlines at every reference as above.

**Field-level override.** `swagger:strfmt` may also sit on a struct
**field** doc to override just that field's published format — e.g.
`// swagger:strfmt int64` on a `uint64` field emits
`{type: string, format: int64}`, a precision-safe, JSON-conformant
string encoding. (By default codescan emits Go-specific integer formats
for unsized/large ints — `uint64` → `{integer, format: uint64}`,
`uint32` → `{integer, format: uint32}`. These vendor formats round-trip
back to Go but are not part of the Swagger 2.0 format set; the
`swagger:strfmt int64` string override is the conformant alternative.)

**Legal keywords.** None at the type level beyond `swagger:strfmt`
itself; the format name is the entire surface.

**Full example.** `fixtures/enhancements/text-marshal/types.go`.

---

## `swagger:enum`

**What it does.** Marks a string-typed (or integer-typed) named type
as an enum and collects the type's `const` declarations.

- **Without `swagger:model`** (the default): the values are applied
  **inline on each model field that references the type** — the
  property gets an `enum` array plus an `x-go-enum-desc` extension
  carrying the per-value godoc descriptions in `<value> <doc-text>`
  shape. The enum type itself is not a standalone definition; the
  values travel with each referencing property.
- **With `swagger:model`**: the enum becomes a **first-class
  definition** carrying the `enum` array (+ `x-go-enum-desc`), and
  referencing fields point at it via `$ref` — the general
  `swagger:model ⇒ definition + $ref` rule applied to enums.

(Edge case: if `swagger:enum` names a type for which no matching
`const` values are found, the enum semantics are dropped and the type
falls through to ordinary type resolution — typically a plain
definition referenced by `$ref`, with no `enum` array.)

**Where it goes.** On a named type declaration. The type's `const`
values are discovered via Go's type-system traversal; they do not
need to live in the same file. The values surface only when a model
reaches the enum type through a field.

**Argument shape.** Optional IDENT naming the type whose `const`
values to collect. On a type declaration the name is redundant, so the
**bare `swagger:enum`** form is accepted and infers the name from the
declared type. `swagger:enum Priority` and a bare `swagger:enum` on
`type Priority …` are equivalent.

**Sample.**

```go
// Priority is the urgency level on a task.
//
// swagger:enum Priority
type Priority string

const (
	// PriorityLow is for tasks that can wait.
	PriorityLow Priority = "low"

	// PriorityMedium is the default.
	PriorityMedium Priority = "medium"

	// PriorityHigh is for tasks that must run soon.
	PriorityHigh Priority = "high"
)

// Task references Priority, which is what makes the enum reachable.
//
// swagger:model
type Task struct {
	Priority Priority `json:"priority"`
}
```

Produces (extract) — the values land on `Task`'s `priority` property,
not on a `Priority` definition:

```json
{
  "Task": {
    "type": "object",
    "properties": {
      "priority": {
        "description": "Priority is the task's urgency.\nlow PriorityLow is for tasks that can wait.\nmedium PriorityMedium is the default.\nhigh PriorityHigh is for tasks that must run soon.",
        "type": "string",
        "enum": ["low", "medium", "high"],
        "x-go-enum-desc": "low PriorityLow is for tasks that can wait.\nmedium PriorityMedium is the default.\nhigh PriorityHigh is for tasks that must run soon."
      }
    }
  }
}
```

By default the const→value mapping is folded into the property's
`description` (as above) **and** duplicated in `x-go-enum-desc`. Set the
scanner option `SkipEnumDescriptions: true` to keep the authored prose as
the description; the mapping then rides `x-go-enum-desc` only. See
[Vendor extensions]({{% relref "/shaping-the-output/vendor-extensions" %}}).

**Legal keywords.** Schema-context keywords. The `enum:` keyword can
ALSO be used inline on the type doc to force a value set; when present,
it overrides the const-derived values and the `x-go-enum-desc` is
recomputed (or dropped) accordingly.

**Full example.** `fixtures/enhancements/enum-overrides/types.go`.

---

## `swagger:allOf`

**What it does.** Marks a struct as participating in an `allOf`
composition. The struct's fields plus any embedded
`swagger:model`-tagged base produce an `allOf: [$ref base, {inline
fields}]` schema. The companion convention is to embed the base
type as an anonymous field with this annotation on the embedding's
doc comment (or on the embedded type itself).

**Where it goes.** On a struct field that embeds another type, or on
a struct type that has at least one embedded base.

**Argument shape.** No args.

**Sample.**

```go
// Animal is the abstract base.
//
// swagger:model
type Animal struct {
	Kind string `json:"kind"`
}

// Dog is an Animal with a breed.
//
// swagger:model
type Dog struct {
	// swagger:allOf
	Animal

	Breed string `json:"breed"`
}
```

Produces:

```json
"Dog": {
  "allOf": [
    {"$ref": "#/definitions/Animal"},
    {
      "type": "object",
      "properties": {
        "breed": {"type": "string", "x-go-name": "Breed"}
      }
    }
  ]
}
```

**Legal keywords.** Schema-context keywords on the inline-object
member (the second `allOf` element).

**Inside a response body.** The same composition applies when the
embedding struct is a `swagger:response` body. The embedded base emits
an `allOf: [{$ref}, …]` arm **only when it is a `swagger:model`** — i.e.
a definition exists to point at. If the embedded type is itself a
`swagger:response` (which has no definition), its fields are inlined
instead of producing a `$ref`.

**Full example.** `fixtures/enhancements/allof-edges/types.go`.

---

## `swagger:alias` — DEPRECATED

**Deprecated.** `swagger:alias` is deprecated and no longer affects the
emitted spec — it is an empty sink that only raises a `validate.deprecated`
diagnostic. (Earlier documentation claimed it published a `$ref` to the
alias target; that was never accurate. Its only real effect was to
force a named **primitive** type to inline its scalar — e.g. `{type:
string}` — instead of producing the `$ref` a named type otherwise gets.
That force-inline behaviour has been removed.)

**Migration.**

- To **inline** a type at a use site, use `swagger:type inline` on the
  field (see [`swagger:type`](#swaggertype)).
- To publish a type as a **first-class definition** that fields `$ref`,
  use `swagger:model`.
- To control alias rendering **globally**, use the `RefAliases` /
  `TransparentAliases` options. A plain (unannotated) Go alias `type T =
  Other` dissolves to its target by default.

**Where it went.** On a type alias / named-type declaration.

**Argument shape.** Optional IDENT (ignored — the annotation has no effect).

---

## `swagger:route`

**What it does.** Declares an HTTP route + operation in one
annotation. The header line carries the method, path, optional tags,
and the operation ID; the comment body carries the operation's
metadata (consumes / produces / schemes / security / parameters /
responses / extensions).

This is the **terser of the two operation-declaration annotations**.
Most go-swagger projects use `swagger:route` for hand-written
operations.

**Where it goes.** On a function or variable declaration whose doc
comment carries the annotation. The Go entity itself doesn't have to
be a handler — the annotation publishes a path/operation independent
of the carrier.

**Argument shape.** Header line:

```
swagger:route <METHOD> <path> [tag1 tag2 …] <operationID>
```

- `<METHOD>` — `GET`, `POST`, `PUT`, `DELETE`, `PATCH`, `HEAD`,
  `OPTIONS`. Case insensitive.
- `<path>` — starts with `/`. Supports path-parameter braces:
  `/items/{id}`. Path templating follows
  [RFC 6570](https://www.rfc-editor.org/rfc/rfc6570) URI Template
  **Level-1 expansion** only (simple `{name}` substitution), as
  required by OpenAPI 2.0. An inline regex constraint written in the
  gorilla/chi style (`/items/{id:[0-9]+}`) is **stripped** to the bare
  `/items/{id}` form and a warning is emitted — OpenAPI 2.0 cannot
  express the constraint, so it is dropped rather than silently failing
  the whole route. The same applies to `swagger:operation`.
- `[tag1 tag2 …]` — optional whitespace-separated list of tags. At
  least two characters each.
- `<operationID>` — the unique operation identifier.

A godoc-style identifier may precede the annotation on the same
comment line:

```go
// ListPets swagger:route GET /pets pets users listPets
```

That leading identifier is recognised as a godoc convention and is
not part of the annotation surface.

**Sample.**

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

**Legal keywords.** All
[body keywords]({{% relref "keywords#body-keywords" %}}) legal in route context
(`consumes`, `produces`, `schemes`, `security`, `parameters`,
`responses`, `extensions`, `externalDocs`) plus inline `deprecated:` and a
body `tags:` list (a string list, unioned and deduplicated with the
header-line tags). The same applies to `swagger:operation`.

The `Parameters:` and `Responses:` sub-languages are documented in
[sub-languages.md §parameters]({{% relref "sub-languages#parameters" %}}) and
[sub-languages.md §responses]({{% relref "sub-languages#responses" %}}).

**Full example.** `fixtures/enhancements/routes-full-petstore-shape/handlers.go`.

---

## `swagger:operation`

**What it does.** Same payload as `swagger:route` but with a
different body shape: instead of the structured `Parameters:` /
`Responses:` keyword surface, `swagger:operation`'s body is a
single YAML document spelling out the OpenAPI operation object
directly.

Use `swagger:operation` when you want to author the operation in
YAML (closer to the OpenAPI spec text) or when the operation has
shapes the keyword surface doesn't cover.

**Where it goes.** Same as `swagger:route` — function or variable
doc comment.

**Argument shape.** Same header shape as `swagger:route`:

```
swagger:operation <METHOD> <path> [tag1 tag2 …] <operationID>
```

**Sample.**

```go
// swagger:operation GET /items/{id} items getItem
//
// ---
// summary: Get item by ID
// parameters:
//   - name: id
//     in: path
//     required: true
//     type: integer
// responses:
//   '200':
//     description: the requested item
//     schema:
//       $ref: '#/definitions/Item'
//   default:
//     $ref: '#/responses/genericError'
func GetItem() {}
```

The `---` delimits the YAML body; everything between the fences is
parsed as an OpenAPI 2.0 operation object.

**Legal keywords.** None inside the YAML body (it's structurally
YAML, not the keyword grammar). The header line is the entire
annotation surface.

**Full example.** `fixtures/enhancements/parameters-map-postdecl/api.go`.

---

## `swagger:parameters`

**What it does.** Declares a Go struct as the parameters set for one
or more operations. Each field of the struct becomes one parameter
on the named operation(s). The field's doc comment carries the
parameter's `in:`, `required:`, validation, and description.

Each parameter's **name** comes from the field's `json:` tag, falling
back to the Go field name when there is no tag. The `form:` tag is not
consulted — add a `json:` tag to control the parameter name (a
`form:"sort_key"` tag alone leaves the name as the Go identifier). A
`name:` keyword in the field doc takes precedence over both, setting the
parameter name explicitly — it is the [universal field-naming
keyword]({{% relref "/maintainers/keywords#name" %}}) and is canonical
here. (The `swagger:name` *annotation* is the legacy form for model
properties and interface methods; in a parameter context it is inert and
now emits a `context-invalid` diagnostic pointing you at `name:`.)

**Where it goes.** On a struct declaration. A bare slice variable
(`var Filters []string`) carries no `in:`/`type:`/`required:` per
field, so it cannot drive parameter generation — parameters must be a
struct.

**Argument shape.** Required IDENTs — the operation IDs this
parameters set applies to. At least one. The same operation ID may
appear in multiple `swagger:parameters` annotations to compose a
parameter set from several structs. Conversely, **one struct may carry
several `swagger:parameters` lines**, each listing a different subset of
operation IDs — the lists accumulate, so a long operation-ID list can be
split across multiple annotation lines for readability.

**Across packages.** The struct need not sit in the same package as the
`swagger:route` or `swagger:operation` it serves. `swagger:parameters`
(and `swagger:response`) declarations are collected across **all scanned
packages** and matched to operations by operation ID — so a shared
parameter set can live in its own package, as long as that package is in
the scan set.

**Sample.**

```go
// ListItemsParams declares pagination + filter parameters for the
// listItems operation.
//
// swagger:parameters listItems
type ListItemsParams struct {
	// Offset is the page offset.
	//
	// in: query
	// minimum: 0
	// default: 0
	Offset int `json:"offset"`

	// Limit is the page size.
	//
	// in: query
	// minimum: 1
	// maximum: 100
	// default: 20
	Limit int `json:"limit"`

	// Tag is the filter tag.
	//
	// in: query
	// required: false
	Tag string `json:"tag,omitempty"`
}
```

**Legal keywords on fields.** [param-context keywords]({{% relref "keywords#parameter-location" %}})
(`in`, `required`, the numeric / length / format validations,
`default`, `example`, `enum`, `allowEmptyValue`, `collectionFormat`).

**Full example.** `fixtures/enhancements/simple-schema-violation/api.go`.

---

## `swagger:response`

**What it does.** Declares a Go struct as a named response object,
emitted into the spec's top-level `responses` map. Routes / operations
reference it by name via the response sub-language (`Responses:`
body in `swagger:route`, or the YAML `$ref` form in
`swagger:operation`).

The struct's fields contribute the response shape:

- A field named `Body` (or carrying `in: body`) becomes the response
  body schema. The body may be a struct, a `$ref`'d model, **or a
  primitive** — e.g. `Body string` emits `schema: {type: string}` and
  `Body []int` emits `schema: {type: array, items: {type: integer}}`.
- Other fields carrying `in: header` become response headers. A field
  with **neither** `Body`/`in: body` nor `in: header` is treated as a
  response **header** by default, not as a body property — so for a body
  schema, name the field `Body` or mark it `in: body`. The header's key
  comes from the `json:` tag / Go field name, or a `name:` keyword in the
  field doc (e.g. `name: X-Rate-Limit`) to set it explicitly.
- An **anonymously embedded** struct marked `in: body` *is* the body —
  the embedded type becomes the body schema (a `$ref` to the model),
  exactly like a named `Body Foo` field, rather than promoting its fields.
  (The same holds for `swagger:parameters`: an `in: body` embed yields the
  single body parameter.)

An `interface{}` / `any`-typed field (or a slice `[]any`) emits an empty
schema — `{}` for a scalar field, `{type: array, items: {}}` for a
slice. An empty schema means "any type" and is valid OpenAPI 2.0; this is
intentional, not a missing type.

**Where it goes.** On a struct declaration.

**Argument shape.** Optional IDENT — the published response name.
Default: the Go type's name.

**Sample.**

```go
// GenericError is the catch-all error response.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		// Message is the human-readable error message.
		Message string `json:"message"`

		// Code is the machine-readable error category.
		Code string `json:"code,omitempty"`
	}

	// X-Request-ID echoes the request correlation header.
	//
	// in: header
	XRequestID string `json:"X-Request-ID"`
}
```

Routes can then reference it via `response:genericError` in their
`Responses:` body.

**Legal keywords on body field.** Schema-context keywords.
**Legal keywords on header field.** Header-context keywords —
numeric / length / format validations, `pattern`, `enum`, `default`,
`example`, `collectionFormat`. `required:` is silently dropped on
headers (the OAS v2 Header object does not carry a `required` field).

**Full example.** `fixtures/enhancements/routes-full-petstore-shape/handlers.go`.

---

## `swagger:ignore`

**What it does.** Excludes the surrounding declaration from the
generated spec. The scanner sees the decl and the doc, classifies
it, then drops it.

**Where it goes.** On a type declaration to exclude the whole type,
or on a struct field doc to exclude that one field.

**Argument shape.** No args.

**Sample (type):**

```go
// Internal is not exposed.
//
// swagger:ignore
type Internal struct {
	SecretField string
}
```

**Sample (field):**

```go
type User struct {
	Name string `json:"name"`

	// PasswordHash is internal.
	//
	// swagger:ignore
	PasswordHash string `json:"-"`
}
```

**Interaction:** when `swagger:ignore` appears AFTER another
classifier on the same comment block (e.g., `swagger:model` first,
then `swagger:ignore`), the first annotation wins and the ignore is
silently overridden. Place `swagger:ignore` first if you genuinely
want the decl excluded.

**Full example.** `fixtures/enhancements/top-level-kinds/types.go`.

---

## `swagger:name`

**What it does.** Overrides the JSON property name that a struct
field or interface method renders as. By default the scanner derives
names from `json:"…"` struct tags (or the Go identifier for fields /
methods with no tag); `swagger:name` overrides that derivation when
the tag-based shape isn't appropriate — typically on **interface
methods**, which cannot carry struct tags.

{{% notice style="note" %}}
`swagger:name` is the **legacy** annotation form. The canonical,
universal field-naming mechanism is the
[`name:` keyword]({{% relref "/maintainers/keywords#name" %}}), which
works at *every* field site — model properties, interface methods,
parameters, and response headers — with the precedence `name:` >
`swagger:name` > `json:` tag > Go field name. `swagger:name` remains
honoured (and idiomatic on interface methods, shown below), but reach
for `name:` in new code; it is the only form that works on parameters
and headers.
{{% /notice %}}

**Where it goes.** On a struct field doc OR an interface method doc.

**Argument shape.** Required IDENT — the JSON property name to use.

**Sample (interface method):**

```go
// UserProfile is the user's profile interface.
//
// swagger:model
type UserProfile interface {
	// ID is the user identifier.
	ID() string

	// FullName is the user's display name.
	//
	// swagger:name fullName
	FullName() string
}
```

Without `swagger:name`, the method `FullName()` would publish as
property `FullName` (PascalCase). The annotation renames it to
`fullName`.

**Legal keywords.** None — the override name is the entire surface.

**Full example.** `fixtures/enhancements/interface-methods/types.go`.

---

## `swagger:type`

**What it does.** Replaces a field's (or named type's) inferred Swagger
type with an **inlined** type. `swagger:type` is an inline directive — it
never emits a `$ref`; the chosen type is rendered directly in place
(the default `$ref`-for-named-types is the *no-annotation* behaviour).

**Where it goes.** On a type declaration, a struct field doc, OR a
`swagger:parameters` field doc.

{{% notice style="note" %}}
**On a parameter field** the override collapses the field to a simple
parameter — useful when a struct- or defined-typed field would otherwise
come out typeless (invalid Swagger 2.0). The argument is restricted to a
**scalar** or a **`[]`-wrapped scalar** there: the `inline` and
type-name forms are rejected with a diagnostic, since a non-body
parameter has no schema to inline a type into. A compatible
`swagger:strfmt` on the same field still rides as a supplementary
format.
{{% /notice %}}

**Argument shape.** Required token, one of:

- a **scalar type** — `string`, `integer`, `number`, `boolean`, `object`
  (or a Go-builtin spelling such as `int64`, `uint32`);
- **`[]T`** — an array whose items are the inlined `T` (recursive:
  `[][]int64`, `[]Custom`);
- **`inline`** — expand the field's own Go type in place, instead of the
  `$ref` a named type would otherwise produce;
- a **known type name** — inline that type's schema (again, no `$ref`).

`array` is **deprecated** — use `inline`, or `[]T` for an explicit element
type; it still works, with a `validate.deprecated` warning. `file` is
rejected with a diagnostic (use [`swagger:file`](#swaggerfile)). An unknown
name falls back to inlining the field's Go type, with a
`validate.unsupported-type` diagnostic.

**Sample (type-level override):**

```go
// ULID is a Crockford-base32 unique identifier rendered as a string.
//
// swagger:type string
type ULID [16]byte
```

Fields typed `ULID` emit as `{type: string}` regardless of the
underlying `[16]byte` shape.

**Sample (field-level override):**

```go
type Document struct {
	// Body is an opaque payload published as a string blob.
	//
	// swagger:type string
	Body json.RawMessage `json:"body"`
}
```

**Interaction with `swagger:strfmt`.** `swagger:type` wins on the type
axis; a `swagger:strfmt` format on the same field is kept only when
**compatible** with the resolved type — a `string` accepts any format,
the numeric types accept the numeric width formats — otherwise it is
dropped with a shape-mismatch diagnostic. `swagger:strfmt` *alone* is
unchanged: it still forces the string-encoded `{type: string, format: …}`.

**Interaction with `swagger:model`.** On a *type declaration* that also
carries `swagger:model`, the override shapes the type's **first-class
definition** (e.g. `swagger:type string` + `swagger:model` → a
`{type: string}` definition) and referencing fields `$ref` it — the
`swagger:model ⇒ definition + $ref` rule. The field-level inline form
above is the behaviour *without* `swagger:model`.

**Full example.** `fixtures/enhancements/named-struct-tags-ref/types.go`.

---

## `swagger:additionalProperties`

**What it does.** Sets a schema's `additionalProperties` — the policy for
keys beyond the named properties. On a struct it **complements** the
named properties; on a map type it **overrides** the element-derived
value schema; on a type that resolved to a bare `$ref` it **defines** a
clean object. See the
[Maps & free-form objects]({{% relref "/tutorials/maps-and-free-form-objects" %}})
tutorial.

**Where it goes.** On a type declaration (alongside `swagger:model`). A
field-level equivalent exists as the
[`additionalProperties:` keyword]({{% relref "/maintainers/keywords#additionalproperties" %}}).

**Argument shape.** Required token, one of:

- **`true`** — allow arbitrary extra keys (`additionalProperties: true`);
- **`false`** — forbid extra keys, closing the object
  (`additionalProperties: false`);
- a **value type** — a primitive / Go-builtin / `[]T`, or a **known type
  name** (which resolves to a `$ref`, and is registered for discovery).
  This reuses the [`swagger:type`](#swaggertype) value grammar, except a
  type name becomes a `$ref` rather than an inline expansion.

**Sample.**

```go
// Settings is an open object: named properties plus typed extra values.
//
// swagger:model
// swagger:additionalProperties integer
type Settings struct {
	Name string `json:"name"`
}
```

**Precedence — lowest priority.** `additionalProperties` only rides on an
`object`. If a prior rule fixed a non-object type (a `swagger:type`
scalar, `swagger:strfmt`, a special type), the marker is dropped with a
`CodeShapeMismatch` diagnostic. It composes with `maxProperties` /
`minProperties` / `patternProperties`. It has no OAS-2 SimpleSchema form,
so it never applies on a non-body parameter or response header.

**Full example.** `fixtures/enhancements/additional-properties/api.go`.

---

## `swagger:patternProperties`

**What it does.** Adds **typed** `patternProperties` entries — each maps a
property-name regex to a value schema. It is the typed counterpart of the
regex-only [`patternProperties:` keyword]({{% relref "/maintainers/keywords#patternproperties" %}})
(which uses an empty, any-value schema).

{{% notice style="note" %}}
`patternProperties` is a JSON-Schema (draft-4) keyword, **beyond the
Swagger 2.0 subset**. codescan emits it ungated — your downstream tooling
must understand it.
{{% /notice %}}

**Where it goes.** On a type declaration (alongside `swagger:model`).

**Argument shape.** A comma-separated list of `"<regex>": <spec>` pairs.
The regex is **double-quoted** (it may contain spaces, colons, commas;
only `\"` is an escape inside it — other backslashes like `\d` are
preserved). Each `<spec>` reuses the value grammar above (primitive /
`[]T` / type-name → `$ref`).

**Sample.**

```go
// Headers carries x-prefixed string values and numeric-keyed counters.
//
// swagger:model
// swagger:patternProperties "^x-": string, "^\d+$": integer
type Headers struct {
	Known string `json:"known"`
}
```

**Precedence.** Same lowest-priority, object-only rule as
`swagger:additionalProperties`. Each regex is RE2-hygiene-checked: one
that does not compile raises a `CodeInvalidAnnotation` warning but is
**preserved**; a structurally malformed pair list is dropped with a
diagnostic.

**Full example.** `fixtures/enhancements/pattern-properties-typed/api.go`.

---

## `swagger:file`

**What it does.** Marks a parameter or response body as a binary file
(`{type: file}`). The scanner emits the file-type marker without
further introspection of the Go type.

**Where it goes.** On a struct field doc inside a
`swagger:parameters` (multipart file upload) or `swagger:response`
(file download) struct.

**Argument shape.** No args.

**Sample.**

```go
// UploadParams declares a multipart file upload.
//
// swagger:parameters uploadFile
type UploadParams struct {
	// File is the uploaded asset.
	//
	// in: formData
	// swagger:file
	File io.ReadCloser `json:"file"`
}
```

**Legal keywords.** Standard parameter / response keywords; the file
marker stacks with `in:` and other parameter shape keywords.

---

## `swagger:default`

**What it does.** Marks the surrounding declaration as the spec's
default value for the corresponding shape. Used in narrow contexts
where the scanner expects an explicit anchor for a default.

This annotation is **value-only** — there's no exported entity it
publishes; it's a classifier hint the scanner consumes during
discovery.

**Where it goes.** On a value declaration (`var`, `const`) or a
struct field.

**Argument shape.** No args.

**Sample.**

```go
// DefaultLimit is the default page size used wherever Limit is not
// supplied by the caller.
//
// swagger:default
var DefaultLimit = 20
```

This annotation has a narrow surface and is not commonly authored
directly. Most spec defaults are carried by the `default:` keyword on
the relevant field.

---

## Annotation × keyword compatibility matrix

A quick orientation for which annotations can carry which keyword
families. See [keywords.md]({{% relref "keywords" %}}) for the per-keyword
contracts.

| Annotation | Numeric/length validations | Schema decorators | `in:` | Meta keywords | `Parameters:` body | `Responses:` body | YAML body |
|------------|----------------------------|-------------------|-------|---------------|--------------------|-------------------|-----------|
| `swagger:meta` | — | — | — | ✅ | — | — | ✅ (security defs, extensions) |
| `swagger:model` | ✅ (on fields) | ✅ | — | — | — | — | — |
| `swagger:strfmt` | — | — | — | — | — | — | — |
| `swagger:enum` | — | (enum keyword via const) | — | — | — | — | — |
| `swagger:allOf` | ✅ (on member fields) | ✅ | — | — | — | — | — |
| `swagger:alias` | — | — | — | — | — | — | — |
| `swagger:route` | — | (deprecated only) | — | (schemes/consumes/produces/security) | ✅ | ✅ | (extensions) |
| `swagger:operation` | — | — | — | — | — | — | ✅ (full op as YAML) |
| `swagger:parameters` | ✅ (on fields) | ✅ (on fields) | ✅ | — | — | — | — |
| `swagger:response` | ✅ (on header fields) | ✅ (on body field) | ✅ (body/header) | — | — | — | — |
| `swagger:ignore` | — | — | — | — | — | — | — |
| `swagger:name` | — | — | — | — | — | — | — |
| `swagger:type` | — | — | — | — | — | — | — |
| `swagger:additionalProperties` | — | ✅ (object schema) | — | — | — | — | — |
| `swagger:patternProperties` | — | ✅ (object schema) | — | — | — | — | — |
| `swagger:file` | — | — | — | — | — | — | — |
| `swagger:default` | — | — | — | — | — | — | — |

A blank cell means the keyword family is not legal in that context;
attempting to use it emits `CodeContextInvalid` and the keyword is
dropped.
