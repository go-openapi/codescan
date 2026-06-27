---
title: "swagger:patternProperties"
weight: 130
description: "Adds typed patternProperties entries mapping a name regex to a value schema."
---


## What it does

Adds **typed** `patternProperties` entries — each maps a property-name
regex to a value schema. It is the typed counterpart of the regex-only
[`patternProperties:` keyword]({{% relref "/maintainers/keywords#patternproperties" %}})
(which uses an empty, any-value schema).

{{% notice style="note" %}}
`patternProperties` is a JSON-Schema (draft-4) keyword, **beyond the
Swagger 2.0 subset**. codescan emits it ungated — your downstream tooling
must understand it.
{{% /notice %}}

## Where it goes

On a type declaration (alongside `swagger:model`).

## Syntax

```ebnf
PatternPropertiesAnnotation = ANN_PATTERN_PROPERTIES , PatternPair , { "," , PatternPair } ;
PatternPair                 = STRING_VALUE , ":" , ValueType ;
ValueType                   = TYPE_REF | IDENT_NAME | "[]" , ValueType ;
```

A comma-separated list of `"<regex>": <spec>` pairs. The regex
(`STRING_VALUE`) is **double-quoted** — it may contain spaces, colons,
commas; only `\"` is an escape inside it, other backslashes like `\d`
are preserved. Each `<spec>` reuses the {{% relref "swagger-type" %}}
value grammar (primitive / `[]T` / type-name → `$ref`).

## Supported keywords

None of its own. It composes with `maxProperties` / `minProperties` /
[`additionalProperties`]({{% relref "swagger-additionalproperties" %}}).

## Example

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
[`swagger:additionalProperties`]({{% relref "swagger-additionalproperties" %}}).
Each regex is RE2-hygiene-checked: one that does not compile raises a
`CodeInvalidAnnotation` warning but is **preserved**; a structurally
malformed pair list is dropped with a diagnostic.

**Full example.** `fixtures/enhancements/pattern-properties-typed/api.go`.
