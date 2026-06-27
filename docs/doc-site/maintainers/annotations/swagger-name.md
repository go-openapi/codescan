---
title: "swagger:name"
weight: 100
description: "Overrides the emitted property name of a struct field or interface method."
---


## What it does

Overrides the JSON property name that a struct field or interface
method renders as. By default the scanner derives names from
`json:"…"` struct tags (or the Go identifier for fields / methods with
no tag); `swagger:name` overrides that derivation when the tag-based
shape isn't appropriate — typically on **interface methods**, which
cannot carry struct tags.

## Where it goes

On a struct field doc OR an interface method doc.

## Syntax

```ebnf
NameAnnotation = ANN_NAME , IDENT_NAME ;
```

The required `IDENT_NAME` is the JSON property name to use.

## Supported keywords

None — the override name is the entire surface.

## Example

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

**Full example.** `fixtures/enhancements/interface-methods/types.go`.

## Deprecated / legacy forms

`swagger:name` is the **legacy** annotation form. The canonical,
universal field-naming mechanism is the
[`name:` keyword]({{% relref "/maintainers/keywords#name" %}}), which
works at *every* field site — model properties, interface methods,
parameters, and response headers — with the precedence `name:` >
`swagger:name` > `json:` tag > Go field name. `swagger:name` remains
honoured (and idiomatic on interface methods, shown above), but reach
for `name:` in new code; it is the only form that works on parameters
and headers.
