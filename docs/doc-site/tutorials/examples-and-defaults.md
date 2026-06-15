---
title: Examples & defaults
weight: 40
description: |
  Attach example values and defaults to properties — and understand the narrow
  swagger:default hint.
---

Example values and defaults are documentation that travels with the schema: an
`example:` shows a caller what a value looks like, a `default:` declares what the
field is when the caller omits it. Both are typed to the field — a numeric
default on an integer field is a JSON number, not a string. The panes below pair
the annotated Go with the fragment the scanner emits, from the test-covered
[`docs/examples/concepts/examples`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/examples)
package.

For the exact value shapes these keywords accept, see
[Keywords]({{% relref "/maintainers/keywords" %}}).

## example

`example: <value>` attaches an `example` to the property, coerced to the field's
type — `Hello, world!` stays a string, `3` becomes a number.

{{< example go="concepts/examples/examples.go" goregion="example"
            json="concepts/examples/testdata/example.json" jsonlabel="#/definitions/Greeting" >}}

The value is not limited to scalars. A **JSON literal** is parsed into a
structured example — a `{ … }` object on a map field, a `[ … ]` array on a slice
field. A bare comma-separated list (`example: a,b`) is *not* split; it is kept
verbatim as a string, so write `example: ["a","b"]` when you need an array.

On a plain `string` field a surrounding pair of double quotes is treated as
**delimiters and stripped** — so `example: "Foo"` yields `Foo`, and
`example: ""` sets an **empty string** (the same applies to `default:`). Bare
values keep their text as-is.

{{< example go="concepts/examples/examples.go" goregion="complexexample"
            json="concepts/examples/testdata/complexexample.json" jsonlabel="#/definitions/Profile" >}}

## default

`default: <value>` sets the property's `default`, again typed to the field — `8080`
is a number, `false` a boolean, `auto` a string.

{{< example go="concepts/examples/examples.go" goregion="default"
            json="concepts/examples/testdata/default.json" jsonlabel="#/definitions/Settings" >}}

## swagger:default

`swagger:default` is a narrow, value-only classifier hint placed on a `var` or
`const`. It does not publish a spec entity of its own — it has no standalone
output — so most spec defaults are carried by the `default:` keyword above
rather than this annotation.

{{< code file="concepts/examples/examples.go" lang="go" region="swaggerdefault" >}}

## On a defined-type field

When a field's type is a named (defined) type, it renders as a `$ref` to that
type's definition — and a `$ref` cannot carry sibling keywords. An `example:` or
`default:` on such a field is therefore preserved on the **override arm of an
`allOf`** compound, so the value still reaches the spec.

{{< example go="concepts/examples/examples.go" goregion="reffield"
            json="concepts/examples/testdata/reffield.json" jsonlabel="#/definitions/Price" >}}

## On a response body

`example:` is not limited to model fields. On a `swagger:response` whose body is
a top-level array (or other non-struct) type, the example lands on the response
body schema:

{{< example go="concepts/examples/examples.go" goregion="responseexample"
            json="concepts/examples/testdata/responseexample.json" jsonlabel="responses[ntpServers]" >}}

## Response examples by media type

A `swagger:operation` YAML body can give a response an `examples:` map keyed by
media type — these populate the OpenAPI response `examples` object:

```go
// swagger:operation GET /status status getStatus
//
// ---
// responses:
//   '200':
//     description: Success
//     examples:
//       application/json:
//         hello: world
```

This per-media-type form is available in the `swagger:operation` YAML body; the
struct-based `swagger:response` does not yet emit per-media-type examples.

## What's next

- [Other type decorators]({{% relref "/tutorials/other-type-decorators" %}}) —
  `readOnly` and `deprecated`.
- [Validations]({{% relref "/tutorials/validations" %}}) — constrain the values
  these examples illustrate.
