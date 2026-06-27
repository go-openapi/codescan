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

If the subtypes are missing from your spec, they are **unreachable**: a subtype
appears only when something references it or you scan with `ScanModels` (the
`-m` flag), the same [reachability rule]({{% relref "type-discovery" %}})
as any model — codescan does not auto-discover subtypes from the base alone.

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
- [`discriminator` keyword]({{% relref "/maintainers/keywords#discriminator" %}})
  and [`swagger:allOf` reference]({{% relref "/maintainers/annotations/swagger-allof" %}}).
