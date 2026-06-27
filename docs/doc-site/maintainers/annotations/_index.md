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

There are seventeen annotations. They divide cleanly by what they
attach to:

- **Spec-level**: `swagger:meta`.
- **Model declarations**: `swagger:model`, `swagger:strfmt`,
  `swagger:enum`, `swagger:allOf`, `swagger:alias`,
  `swagger:additionalProperties`, `swagger:patternProperties`.
- **Operation declarations**: `swagger:route`, `swagger:operation`.
- **Companion declarations**: `swagger:parameters`, `swagger:response`.
- **Local hints**: `swagger:ignore`, `swagger:name`, `swagger:type`,
  `swagger:file`, `swagger:default`.

This section is the **author-first reference**. Each annotation has its
own page covering what it produces, where it goes, its EBNF-like
syntax, the keywords legal inside its block, and at least one worked
example. Browse them below (sorted alphabetically), or start from the
[Annotation index]({{% relref "/annotation-index" %}}) for the one-row-each
overview.

{{< children type="card" description="true" >}}

For the per-keyword reference, see [keywords.md]({{% relref "keywords" %}}).
For the embedded sub-languages (`Parameters:` and `Responses:` body
grammars, YAML extensions, etc.), see
[sub-languages.md]({{% relref "sub-languages" %}}). For the formal grammar,
see [grammar.md]({{% relref "grammar" %}}).

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
  per-annotation pages for the exact rules.

## Annotation × keyword compatibility matrix

A quick orientation for which annotations can carry which keyword
families. See [keywords.md]({{% relref "keywords" %}}) for the per-keyword
contracts, and each annotation's own page for the detail.

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
