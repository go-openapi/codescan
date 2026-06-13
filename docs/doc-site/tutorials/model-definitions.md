---
title: Model definitions
weight: 10
description: |
  Turn Go types into spec definitions — structs, string formats, enums, allOf
  composition, and the per-type overrides.
---

A `definitions` entry is the most common thing you ask codescan to produce. This
page walks the annotations that create and shape one, from the plain
`swagger:model` struct to the per-type overrides. Each pane pairs the annotated
Go (left) with the exact fragment the scanner emits (right) — both come from the
test-covered [`docs/examples/concepts/models`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/models)
package.

For the exhaustive rule on any annotation below, follow its link to the
[Maintainers reference]({{% relref "/maintainers/annotations" %}}).

## swagger:model

`swagger:model` publishes a Go struct as a definition. Field doc comments become
property descriptions; `json` tags drive the property names; the Go type drives
the JSON-Schema `type` / `format`.

{{< example go="concepts/models/models.go" goregion="model"
            json="concepts/models/testdata/model.json" jsonlabel="#/definitions/Pet" >}}

## swagger:strfmt

`swagger:strfmt <name>` marks a named string type as a custom format. The type
does not become its own definition — instead, every field typed by it renders as
`{type: string, format: <name>}`.

{{< example go="concepts/models/models.go" goregion="strfmt"
            json="concepts/models/testdata/strfmt.json" jsonlabel="#/definitions/Device" >}}

## swagger:enum

`swagger:enum <name>` collects the type's `const` values. When a model field
references the type, the property carries the `enum` array and an
`x-go-enum-desc` extension built from the per-value doc comments. (The enum type
is reachable, and so emitted, only because a model field points at it.)

{{< example go="concepts/models/models.go" goregion="enum"
            json="concepts/models/testdata/enum.json" jsonlabel="#/definitions/Task" >}}

## swagger:allOf

Embedding base types under `swagger:allOf` composes a schema. Each embedded base
becomes a `$ref` arm of the `allOf`; the struct's own (non-embedded) fields form
a final inline arm. That last arm is inline rather than a `$ref` because those
fields are new to this type — they belong to no existing definition to point at.
Here `Dog` embeds two bases (`Animal`, `Tagged`) and adds `breed`, producing
three arms: two `$ref`s and one inline object.

{{< example go="concepts/models/models.go" goregion="allof"
            json="concepts/models/testdata/allof.json" jsonlabel="#/definitions/Dog" >}}

## swagger:type

`swagger:type <type>` overrides the type codescan would infer. Here a `[16]byte`
field is published as a `string`.

{{< example go="concepts/models/models.go" goregion="type"
            json="concepts/models/testdata/type.json" jsonlabel="#/definitions/Token" >}}

The accepted values are the scalar Swagger types — `string`, `integer`,
`number`, `boolean`, `object` (plus the Go builtin names codescan resolves).
`array` and `file` are not accepted here; an unrecognized value leaves the field
on its underlying Go type.

## swagger:name

A model defined as an **interface** publishes one property per nullary method.
By default the property name is the camelCased method name — so `Maker()`
already becomes `maker` with no annotation. `swagger:name <name>` is the
**override** for when that default is not what you want (interface methods
cannot carry a `json` tag). Here `StructType()` would default to `structType`;
the annotation publishes it as `jsonClass` instead.

{{< example go="concepts/models/models.go" goregion="name"
            json="concepts/models/testdata/name.json" jsonlabel="#/definitions/Car" >}}

## swagger:ignore

`swagger:ignore` drops a declaration from the output. The scanner sees `Secret`,
classifies it, then excludes it — so it never reaches the definitions (a fact
the example's `TestIgnoreOmitsType` asserts).

{{< code file="concepts/models/models.go" lang="go" region="ignore" >}}

## What's next

- [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}) — wire
  these models into paths, parameters and responses.
- [Validations]({{% relref "/tutorials/validations" %}}) — constrain field values
  with keyword annotations.
- [Shaping the output]({{% relref "/shaping-the-output" %}}) — alias handling,
  `$ref` vs inline, nullable pointers and more.
