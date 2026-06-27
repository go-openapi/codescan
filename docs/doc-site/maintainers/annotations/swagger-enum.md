---
title: "swagger:enum"
weight: 50
description: "Marks a named type as an enum and collects its const values."
---


## What it does

Marks a string-typed (or integer-typed) named type as an enum and
collects the type's `const` declarations.

- **Without `swagger:model`** (the default): the values are applied
  **inline on each model field that references the type** — the property
  gets an `enum` array plus an `x-go-enum-desc` extension carrying the
  per-value godoc descriptions in `<value> <doc-text>` shape. The enum
  type itself is not a standalone definition.
- **With `swagger:model`**: the enum becomes a **first-class definition**
  carrying the `enum` array (+ `x-go-enum-desc`), and referencing fields
  point at it via `$ref` — the general `swagger:model ⇒ definition + $ref`
  rule applied to enums.

If `swagger:enum` names a type for which no matching `const` values are
found, the enum semantics are dropped and the type falls through to
ordinary type resolution (typically a plain `$ref`, no `enum` array).

## Where it goes

On a named type declaration. The type's `const` values are discovered via
Go's type-system traversal; they do not need to live in the same file.
The values surface only when a model reaches the enum type through a field.

## Syntax

```ebnf
EnumBlock = ANN_ENUM , [ IDENT_NAME ] , [ Title ] , [ Description ] ;
```

The optional `IDENT_NAME` names the type whose `const` values to collect.
On a type declaration the name is redundant, so the bare `swagger:enum`
form is accepted and infers the name from the declared type:
`swagger:enum Priority` and a bare `swagger:enum` on `type Priority …`
are equivalent.

## Supported keywords

[Schema-context keywords]({{% relref "keywords#schema-decorators" %}}). The
`enum:` keyword can ALSO be used inline on the type doc to force a value
set; when present, it overrides the const-derived values and the
`x-go-enum-desc` is recomputed (or dropped) accordingly.

## Example

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

Produces (extract) — the values land on `Task`'s `priority` property, not
on a `Priority` definition:

```json
{
  "Task": {
    "type": "object",
    "properties": {
      "priority": {
        "type": "string",
        "enum": ["low", "medium", "high"],
        "x-go-enum-desc": "low PriorityLow is for tasks that can wait.\nmedium PriorityMedium is the default.\nhigh PriorityHigh is for tasks that must run soon."
      }
    }
  }
}
```

By default the const→value mapping is folded into the property's
`description` **and** duplicated in `x-go-enum-desc`. Set the scanner
option `SkipEnumDescriptions: true` to keep the authored prose as the
description; the mapping then rides `x-go-enum-desc` only. See
[Vendor extensions]({{% relref "vendor-extensions" %}}).

**Full example.** `fixtures/enhancements/enum-overrides/types.go`.
