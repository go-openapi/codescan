---
title: Maps & free-form objects
weight: 13
description: |
  How Go maps render as objects, which key types survive, and how to control a
  schema's open/closed/typed extra keys with additionalProperties and
  patternProperties.
---

Not every object has a fixed set of named fields. A Go `map` models an object
with **dynamic keys**; a struct can be marked **open** (extra keys allowed),
**closed** (extra keys forbidden), or given a **typed** value schema for its
extras. codescan expresses all of this with `additionalProperties` and
`patternProperties`. The panes below are rendered from the test-covered
[`docs/examples/concepts/maps`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/maps)
package.

## Maps become objects

A map field renders as `{type: object}` whose values all share one schema,
carried as `additionalProperties` — the value schema is derived from the Go map's
element type:

{{< example go="concepts/maps/maps.go" goregion="naturalmap"
            json="concepts/maps/testdata/naturalmap.json" jsonlabel="#/definitions/Inventory" >}}

## Which map keys work

A map key has to become a JSON object key — a string. codescan accepts every key
type `encoding/json` can stringify: **string** kinds, every **integer / unsigned**
kind (so `map[int]V`, `map[uint8]V`, …), and any type implementing
`encoding.TextMarshaler`. An integer-keyed map is therefore still a valid object:

{{< example go="concepts/maps/maps.go" goregion="keytypes"
            json="concepts/maps/testdata/keytypes.json" jsonlabel="#/definitions/Lookups" >}}

{{% notice style="info" %}}
A key JSON cannot stringify — a `float`, `bool`, a struct without
`TextMarshaler`, or an interface — is **not** silently dropped: the map's
`additionalProperties` is omitted and a `CodeUnsupportedType` diagnostic
("additionalProperties dropped") is raised. A `json:"-"` map field is muted
before this check, so it never warns.
{{% /notice %}}

## Open & closed objects

A struct normally renders with just its named properties and says nothing about
extra keys. The decl-level `swagger:additionalProperties <spec>` marker decides
the policy, where `<spec>` is `true`, `false`, or a value type:

{{< code file="concepts/maps/maps.go" lang="go" region="marker" >}}

`false` **closes** the object (only the named properties are valid); `true`
**opens** it (any extra key is allowed):

{{< compare left="concepts/maps/testdata/closed.json"  leftlabel="false — closed object"
            right="concepts/maps/testdata/open.json"   rightlabel="true — open object" >}}

A **type spec** instead of a bool gives the extra values a schema — a primitive
(or `[]T`), or a model name that becomes a `$ref` (the same value-type grammar as
[`swagger:type`]({{% relref "/maintainers/annotations#swaggertype" %}}), except a
type name resolves to a `$ref`):

{{< compare left="concepts/maps/testdata/typed.json"  leftlabel="integer — typed values"
            right="concepts/maps/testdata/ref.json"   rightlabel="Thing — model values" >}}

On a **map** type the marker *overrides* the element-derived value schema; on a
struct it *complements* the named properties (as above). It composes with the
object validations `maxProperties` / `minProperties` / `patternProperties`.

{{% notice style="info" %}}
The model name is a **bare leaf**: codescan resolves it in the annotating type's
own package first, then uniquely across the scanned model set, so a value type
declared in another package resolves to a `$ref` by name. A leaf that matches a
model in several packages is ambiguous — it is dropped with a
`validate.ambiguous-type-name` diagnostic. See
[Resolving $ref name conflicts]({{% relref "/shaping-the-output/resolving-name-conflicts#referencing-a-model-by-leaf-across-packages" %}}).
{{% /notice %}}

## Per-field control

The same `<spec>` is available as a field keyword,
`additionalProperties: <spec>`, decorating one struct field. On a map field it
overrides the value schema; on a `$ref`'d field the value rides an `allOf`
sibling so the reference is preserved:

{{< example go="concepts/maps/maps.go" goregion="fieldkeyword"
            json="concepts/maps/testdata/fieldkeyword.json" jsonlabel="#/definitions/Holder" >}}

## Pattern properties

`patternProperties` constrains *extra* keys by a name regex rather than allowing
all of them. The regex-only
[`patternProperties:` keyword]({{% relref "/maintainers/keywords#patternproperties" %}})
maps a pattern to an empty (any-value) schema. The decl-level
`swagger:patternProperties "<re>": <spec>, …` marker is the **typed** counterpart
— each quoted regex pairs with a value spec (a primitive, or a model name that
becomes a `$ref`):

{{< example go="concepts/maps/maps.go" goregion="patterntyped"
            json="concepts/maps/testdata/patterntyped.json" jsonlabel="#/definitions/TypedPatterns" >}}

{{% notice style="note" %}}
**Beyond Swagger 2.0.** `patternProperties` is a JSON-Schema (draft-4) keyword,
not part of the Swagger 2.0 Schema Object subset. codescan emits it **ungated**,
consistent with go-openapi's JSON-Schema-first stance — your downstream tooling
must understand it. Each regex is RE2-hygiene-checked: one that does not compile
raises a `CodeInvalidAnnotation` warning but is **preserved** on the schema.
{{% /notice %}}

{{% notice style="info" %}}
`additionalProperties` and `patternProperties` are **object-schema** keywords —
they only ride on an object. They are the *lowest-priority* annotations: if a
prior rule already fixed a non-object type (a `swagger:type` scalar, a
`swagger:strfmt`, a special type), the marker is dropped with a
`CodeShapeMismatch` diagnostic. Object schemas also have no OAS-2 SimpleSchema
form, so neither keyword applies on a non-body parameter or a response header
(it is dropped there with a diagnostic).
{{% /notice %}}

## What's next

- [Validations]({{% relref "/tutorials/validations" %}}) — `maxProperties` /
  `minProperties` and the regex-only `patternProperties:` keyword.
- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) — the
  `$ref` mechanics the typed value schemas rely on.
