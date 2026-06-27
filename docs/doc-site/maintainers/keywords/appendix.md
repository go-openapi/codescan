---
title: "Appendix: shapes & contexts"
weight: 60
description: "Reference tables — the value shapes the lexer classifies, and the meaning of each annotation-context token."
---

The two reference tables behind the keyword class pages: the **value shapes** the
lexer assigns to a keyword's value, and the **context tokens** used in each page's
scoped summary table.

## Value shapes

The grammar's lexer classifies every value into one of these shapes. The shape
determines which Walker callback fires for the property and which field of
`Property.Typed` carries the parsed value.

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
`CodeInvalidNumber` / `CodeInvalidInteger` / `CodeInvalidBoolean` diagnostic and the
property reaches the Walker with a zero-value payload. Consumers gate on
`Property.IsTyped()` to skip malformed-typed values; the corresponding builder field
stays unwritten.

## Annotation contexts

The closed set of contexts a keyword can legally appear in. Each class page's scoped
summary table combines these in its **Contexts** column.

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

Using a keyword outside its legal contexts emits a `CodeContextInvalid` diagnostic
and the keyword is dropped from the affected block. The
[Context matrix]({{% relref "_index#context-matrix" %}}) maps these tokens onto the
annotation families.
