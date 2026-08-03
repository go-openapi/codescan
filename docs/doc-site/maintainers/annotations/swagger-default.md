---
title: "swagger:default"
weight: 40
description: "Deprecated no-op — defaults are carried by the default: keyword, or a default response code."
---


{{% notice style="warning" %}}
**Deprecated.** `swagger:default` never emitted a `default` into the spec, in any
placement or form. It is now an empty sink that only raises a
`validate.deprecated` diagnostic. Use the
[`default:` keyword]({{% relref "/maintainers/keywords/schema-validations-and-decorators#default" %}}),
or a `default` response code in a route's `Responses:` body.
{{% /notice %}}

## Usage

```goish
// swagger:default [ VALUE ]
```

## What it does

Nothing. It is parsed, reported as deprecated, and ignored.

Previously it also **suppressed** the schema of a named basic type it was placed
on: the classifier claimed the target without writing it, so the declared type
published a typeless definition and every field referencing it emitted a typeless
property, silently. That is fixed — an annotated type now emits exactly what it
would emit unannotated.

## Why it was retired

Every place OpenAPI 2.0 admits a `default` is already served, so the annotation
had no meaning left to implement:

| Where a default can appear | How to write it |
|---|---|
| Schema object — a model field, or a type declaration | [`default:` keyword]({{% relref "/maintainers/keywords/schema-validations-and-decorators#default" %}}) |
| Parameter object (non-body) | `default:` keyword |
| Items object | `default:` keyword |
| Header object | `default:` keyword |
| Responses object — an operation's default response | `default:` as the response code in a `Responses:` body |

The keyword's context set is exactly the list of OAS 2.0 objects that carry a
`default`; the response-code head closes the remainder.

## Where it goes

Anywhere it used to — the annotation is still recognised so existing source keeps
scanning. It has no effect wherever it appears.

## Grammar (EBNF)

```ebnf
DefaultClassifierBlock = ANN_DEFAULT , [ VALUE ] , [ Title ] , [ Description ] ;
```

The value argument is optional and unread. It used to be mandatory, which made
the bare form this page once documented a hard parse error.

## Supported keywords

None.

## Example

Replace it with the keyword:

```go
// Port is the listen port.
//
// swagger:model Port
// default: 8080
type Port int
```

For an operation's default response, use the response code:

```go
// swagger:route GET /things things listThings
//
// Responses:
//   200: thingList
//   default: genericError
```
