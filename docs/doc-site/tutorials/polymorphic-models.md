---
title: Polymorphic models
weight: 15
description: |
  Model a Swagger 2.0 type hierarchy — a base type with a discriminator and
  subtypes that compose it with swagger:allOf.
---

Swagger 2.0 expresses polymorphism with three ingredients:

1. a **base** type that declares a **`discriminator`** — the property whose value
   says which concrete subtype a payload is;
2. **subtypes** that include the base via `allOf` and add their own fields;
3. a discriminator **value** per subtype (here, the subtype's definition name).

The panes below are backed by the test-covered
[`docs/examples/concepts/polymorphism`](https://github.com/go-openapi/codescan/tree/master/docs/examples/concepts/polymorphism)
package.

## The base type

Mark one property `discriminator: true`. codescan writes that property's **name**
onto the schema's `discriminator`. A discriminator property must also be
`required` — a consumer cannot pick a subtype from a value that may be absent.

{{< example go="concepts/polymorphism/polymorphism.go" goregion="base"
            json="concepts/polymorphism/testdata/base.json" jsonlabel="#/definitions/Pet" >}}

## The subtypes

Each subtype embeds the base as an anonymous field annotated `swagger:allOf`. The
result is `allOf: [ {$ref to the base}, {the subtype's own fields} ]` — the same
composition covered in [Model definitions]({{% relref "/tutorials/model-definitions" %}}#swaggerallof),
now given meaning by the base's discriminator.

{{< example go="concepts/polymorphism/polymorphism.go" goregion="children"
            json="concepts/polymorphism/testdata/subtype.json" jsonlabel="#/definitions/Cat" >}}

`Dog` follows the identical shape. A payload is then recognised as a `Cat` or a
`Dog` by its `petType` value.

## How subtypes are discovered

A family has an awkward property: the references all point **upwards**. A subtype
`$ref`s its base, and nothing ever `$ref`s a subtype. So an API that returns the
base — the whole point of polymorphism — names only the base, and the ordinary
[reachability rule]({{% relref "type-discovery" %}}) would stop right there,
leaving a `discriminator` with nothing to discriminate between.

codescan therefore looks the relation up **backwards**. When a definition that
declares a `discriminator` enters the spec, every `swagger:model` that composes it
under `swagger:allOf` is pulled in with it — wherever those subtypes are declared,
including other packages. **No `ScanModels` needed.** The route below references
only `Pet`:

{{< example go="concepts/polymorphism/polymorphism.go" goregion="route"
            json="concepts/polymorphism/testdata/spec-reachable-only.json"
            jsonlabel="Whole spec, scanned WITHOUT ScanModels" >}}

`Cat` and `Dog` are in there, and each pull is announced on the
[`OnDiagnostic`](https://pkg.go.dev/github.com/go-openapi/codescan#Options) sink,
so a definition you did not ask for by name is never a mystery:

{{< code file="concepts/polymorphism/testdata/hints.txt" lang="text" >}}

Three consequences worth knowing:

- **Reachability, not existence.** The trigger is the base *entering the spec*,
  not merely existing in the scanned source. A discriminated base that nothing
  references still emits nothing — bases are not roots, or every hierarchy in a
  shared library would land in every spec.
- **The family travels as a unit.** With
  [`PruneUnusedModels`]({{% relref "pruning-unused-models" %}}) a reachable
  discriminated base keeps its subtypes, even though no `$ref` reaches them; and
  an unreachable base is dropped *together with* its subtypes. You never get a
  base whose subtypes have vanished.
- **Only `swagger:allOf` counts.** A plain embed inlines the base's properties
  instead of composing them, so it is not a subtype relation. The
  [`DefaultAllOfForEmbeds`]({{% relref "composing-embeds-with-allof" %}}) option
  changes how embeds *render*, deliberately not which definitions *exist*.

## Multi-level hierarchies

A subtype can be a base in its own right. Write the intermediate level as an
**interface** — only an interface can be embedded by the concrete structs beneath
it — composing its parent with `swagger:allOf` and declaring a discriminator of
its own:

{{< example go="concepts/polymorphism-nested/nested.go" goregion="hierarchy"
            json="concepts/polymorphism-nested/testdata/intermediate.json"
            jsonlabel="#/definitions/Polygon — subtype AND base" >}}

Discovery cascades: the route references `Shape`, `Shape` pulls `Polygon`, and
`Polygon` — itself only just discovered — pulls `Square`. Note **where each
level's discriminator lands**, because the two differ:

| level | shape | `discriminator` |
|-------|-------|-----------------|
| root (`Shape`) | a plain object | at the **top level** |
| intermediate (`Polygon`) | `allOf: [ $ref Shape, {own} ]` | inside its **own `allOf` member** |
| leaf (`Square`) | `allOf: [ $ref Polygon, {own} ]` | none of its own |

A leaf never inherits its base's `discriminator` as its own: it points at a
discriminated base, which is what makes it a *subtype*, not a base.

{{% notice style="info" %}}
The discriminator **value** for each subtype is its definition name (`Cat`,
`Dog`) — so `petType` must carry exactly `"Cat"` or `"Dog"`. codescan does not
implement a custom-value annotation (`swagger:discriminatorValue`), so the
subtype name is the value. Keep the discriminator a plain `string` and `required`
on the base; it is inherited by every subtype through the `$ref`.
{{% /notice %}}

## What's next

- [Model definitions]({{% relref "/tutorials/model-definitions" %}}) — the
  `swagger:allOf` composition this builds on, and the rest of the model surface.
- [Routes & operations]({{% relref "/tutorials/routes-and-operations" %}}) —
  return a base type and let the discriminator carry the subtype.
- [`discriminator` keyword]({{% relref "/maintainers/keywords/schema-validations-and-decorators#discriminator" %}})
  and [`swagger:allOf` reference]({{% relref "/maintainers/annotations/swagger-allof" %}}).
