---
title: "swagger:ignore"
weight: 70
description: "Excludes the surrounding declaration (or one field) from the generated spec."
---

## Usage

```goish
// swagger:ignore
```

## What it does

Excludes the surrounding declaration from the generated spec.

The scanner sees the decl and the doc, classifies it, then drops it.

When `swagger:ignore` appears **after** another classifier on the same
comment block (e.g. `swagger:model` first, then `swagger:ignore`), the
first annotation wins and the ignore is silently overridden. Place
`swagger:ignore` first if you genuinely want the decl excluded.

## Where it goes

On a type declaration to exclude the whole type, or on a struct field
doc to exclude that one field.

## Grammar (EBNF)

```ebnf
IgnoreBlock = ANN_IGNORE , [ Title ] , [ Description ] ;
```

Takes no argument — an optional title/description may follow on the
doc comment.

## Supported keywords

None — the annotation is a stateless classifier marker.

## Example

`swagger:ignore` produces no schema, so there is no live spec pane here: the
type below is scanned, classified, then dropped — it never reaches
`definitions`. On a type it excludes the whole declaration; on a struct
field it excludes just that one property (e.g. a `PasswordHash` kept out of
the wire shape).

{{< code file="concepts/models/models.go" region="ignore" lang="go" >}}

**Full example.** `fixtures/enhancements/top-level-kinds/types.go`.
