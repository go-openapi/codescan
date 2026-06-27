---
title: "swagger:ignore"
weight: 70
description: "Excludes the surrounding declaration (or one field) from the generated spec."
---


## What it does

Excludes the surrounding declaration from the generated spec. The
scanner sees the decl and the doc, classifies it, then drops it.

When `swagger:ignore` appears **after** another classifier on the same
comment block (e.g. `swagger:model` first, then `swagger:ignore`), the
first annotation wins and the ignore is silently overridden. Place
`swagger:ignore` first if you genuinely want the decl excluded.

## Where it goes

On a type declaration to exclude the whole type, or on a struct field
doc to exclude that one field.

## Syntax

```ebnf
IgnoreBlock = ANN_IGNORE , [ Title ] , [ Description ] ;
```

Takes no argument — an optional title/description may follow on the
doc comment.

## Supported keywords

None — the annotation is a stateless classifier marker.

## Example

```go
// Internal is not exposed.
//
// swagger:ignore
type Internal struct {
	SecretField string
}
```

On a single field:

```go
type User struct {
	Name string `json:"name"`

	// PasswordHash is internal.
	//
	// swagger:ignore
	PasswordHash string `json:"-"`
}
```

**Full example.** `fixtures/enhancements/top-level-kinds/types.go`.
