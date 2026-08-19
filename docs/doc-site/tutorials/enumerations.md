---
title: Enumerations
weight: 12
description: |
  Publish a Go const block as a spec enum — any constant expression, the type
  and format taken from the declaration, and the same members inline on
  parameters and headers.
---

An enum in Go is a named type plus a block of constants. `swagger:enum` turns
that pair into an `enum` array on every schema, parameter and header the type
reaches. This page covers what the scanner accepts on the value side, what
decides the emitted `type` / `format`, and the two shapes that do not work.

{{% notice style="note" title="`swagger:enum` and `enum:` are two different things" %}}
They produce the same spec keyword from opposite directions, and the names are
close enough to trip over:

| | `swagger:enum` — an **annotation** | `enum:` — a **keyword** |
|---|---|---|
| Where | on the type declaration | inside any annotation block, on a field, parameter, header or declaration |
| Members come from | the Go `const` block of that type, read from the type-checker | the literal list you write after the colon |
| Type / format | the **declared Go type** | the schema the keyword sits on |
| Use it when | the values already exist as Go constants | there is no const block, or the members are not Go values at all |

```go
// swagger:enum Kind        ← annotation: members are collected from the consts
type Kind string
const (
	KindA Kind = "a"
	KindB Kind = "b"
)

type Filter struct {
	// enum: asc, desc       ← keyword: members are taken verbatim
	Order string `json:"order"`
}
```

The annotation is the better tool whenever the constants exist: it stays in
sync with the code, carries each member's doc comment into `x-go-enum-desc`,
and cannot drift from the Go values. The keyword is the escape hatch for
everything else.
{{% /notice %}}

Every Go snippet below comes from the test-covered
[`docs/examples/concepts/enums`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/enums)
package, and every JSON pane is a golden file a test regenerates.

## swagger:enum

`swagger:enum <name>` collects the `const` values declared with that type. A
bare `swagger:enum` on the type declaration works too — the name is inferred
from the declaration it sits on.

The enum type is emitted **because something points at it**: a model field, a
parameter, a header. On its own it is unreachable, and unreachable types are not
published. Add `swagger:model` to the enum type to make it a first-class
definition (carrying the `enum` array) that fields `$ref` instead — the general
`swagger:model ⇒ definition + $ref` rule.

Each member's doc comment becomes a line of the `x-go-enum-desc` extension, and
is appended to the property description. Set
[`SkipEnumDescriptions`]({{% relref "options-reference" %}}) to keep the
mapping on the extension only.

## Any constant expression, not just literals

The values come from the Go **type-checker**, which has already evaluated the
const block. So the members do not have to be written as literals — anything the
compiler can fold is collected.

`iota` is the case that matters most, because after the first line there is
nothing left in the source to read: the following specs carry neither a type nor
a value, and inherit both implicitly.

{{< example go="concepts/enums/enums.go" goregion="iota"
            json="concepts/enums/testdata/iota.json" jsonlabel="#/definitions/Schedule" >}}

Constant expressions and references to earlier members are collected the same
way.

{{< example go="concepts/enums/enums.go" goregion="expressions"
            json="concepts/enums/testdata/expressions.json" jsonlabel="#/definitions/Threshold" >}}

The same goes for every other constant form Go offers: negative values, the
non-decimal bases (`0x2a`, `0b101010`, `0o52`) and digit separators, values
above `MaxInt64` in an unsigned enum, `true` / `false`, rune literals, and both
the raw and the escaped string forms.

Negative members are worth a pane of their own, since a signed constant is not a
literal in the Go grammar — it is an expression wrapping one:

{{< example go="concepts/enums/enums.go" goregion="signed"
            json="concepts/enums/testdata/signed.json" jsonlabel="#/definitions/Camera" >}}

## The type comes from the declaration

`type` and `format` are read from the Go type you declared, never from the
members. `PanDirection` above is an `int8`, so the property is
`{integer, int8}` — even though every member would fit in a smaller or larger
box.

This is also what makes the const block **safe to reorder**. `Zoom` is a
`float32` whose first member is written `0`, an integer literal; the schema is a
number enum regardless of which member comes first:

{{< example go="concepts/enums/enums.go" goregion="width"
            json="concepts/enums/testdata/width.json" jsonlabel="#/definitions/Lens" >}}

A type declared over another **named** type keeps what that type contributed. An
enum written over a string format is still that format:

{{< example go="concepts/enums/enums.go" goregion="strfmt"
            json="concepts/enums/testdata/strfmt.json" jsonlabel="#/definitions/Label" >}}

## Parameters and headers

OpenAPI 2.0 forbids a `$ref` on a non-body parameter or a response header, so
there the members and the format are written **inline**. Nothing changes on the
annotation side — the same enum type reaches `in: query`, `path`, `header` and
`formData`, and the `items` of an array-typed one:

{{< example go="concepts/enums/enums.go" goregion="params"
            json="concepts/enums/testdata/params.json" jsonlabel="GET /cameras/search — parameters"
            full="concepts/enums/testdata/full.json" >}}

## Two shapes that do not work

**A `rune` or `byte` enum emits integers.** It is collected like any other, and
`'a'` reaches the spec as `97`:

{{< example go="concepts/enums/enums.go" goregion="runes"
            json="concepts/enums/testdata/runes.json" jsonlabel="#/definitions/Glyph" >}}

That is unlikely to be what you pictured, and it is the only faithful answer: a
scalar `rune` is an `int32` on the wire as much as in Go, so `json.Marshal`
writes `97`, and `encoding/json` **refuses** to unmarshal `"a"` back into the
field. A string-typed schema would describe a payload your own server rejects.
If you meant characters, declare the type over `string`
(`type Letter string`, `LetterA Letter = "a"`) — that changes the wire, and the
schema follows.

**An alias to a basic type cannot host an enum.**

```go
type Unsigned = uint64          // an alias, not a new type

// swagger:enum Unsigned        // ← collects nothing
const Zero Unsigned = 0
```

The Go type-checker erases the alias, so `Zero` is indistinguishable from any
other `uint64` constant and there is no set of members to collect. Declare a
real type instead (`type Unsigned uint64`). An alias to a *named* enum type is
fine — the named type survives.

## Where to go next

- [Validations]({{% relref "/tutorials/validations" %}}) — the other constraints
  a property can carry, and the reduced surface parameters accept.
- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) — the rest
  of the per-type annotations.
- [`swagger:enum` reference]({{% relref "/maintainers/annotations/swagger-enum" %}})
  — the exhaustive rule.
