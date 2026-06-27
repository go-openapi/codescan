---
title: "swagger:name"
weight: 100
description: "Overrides the emitted property name of a struct field or interface method."
---

## Usage

```goish
// swagger:name IDENT_NAME
```

## What it does

Overrides the JSON property name that a struct field or interface method
renders as.

By default the scanner derives names from `json:"…"` struct tags (or the
Go identifier for fields / methods with no tag); `swagger:name` overrides
that derivation when the tag-based shape isn't appropriate — typically on
**interface methods**, which cannot carry struct tags.

## Where it goes

On a struct field doc OR an interface method doc.

## Grammar (EBNF)

```ebnf
NameAnnotation = ANN_NAME , IDENT_NAME ;
```

The required `IDENT_NAME` is the JSON property name to use.

## Supported keywords

None — the override name is the entire surface.

## Example

On an interface method, `swagger:name` overrides the property name the
method would otherwise publish under (PascalCase Go method name) with the
chosen JSON name:

{{< example
    go="concepts/models/models.go" goregion="name"
    json="concepts/models/testdata/name.json" >}}

**Full example.** `fixtures/enhancements/interface-methods/types.go`.

## Deprecated / legacy forms

`swagger:name` is the **legacy** annotation form. The canonical,
universal field-naming mechanism is the
[`name:` keyword]({{% relref "/maintainers/keywords/parameters-and-responses#name" %}}), which
works at *every* field site — model properties, interface methods,
parameters, and response headers — with the precedence `name:` >
`swagger:name` > `json:` tag > Go field name. `swagger:name` remains
honoured (and idiomatic on interface methods, shown above), but reach
for `name:` in new code; it is the only form that works on parameters
and headers.
