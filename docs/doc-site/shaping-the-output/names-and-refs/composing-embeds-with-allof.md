---
title: Composing embeds with allOf
weight: 35
description: |
  Render a plain struct embed as an allOf composition — a $ref to the embedded
  model plus a sibling member for the embedding struct's own fields — instead of
  inlining the promoted properties, with DefaultAllOfForEmbeds.
---

When a struct embeds another struct, Go *promotes* the embedded fields, and by
default codescan mirrors that: the embedded type's properties are inlined flat
into the embedding schema. That is faithful to the Go value, but it loses the
*"this composes `Base`"* relationship — every embedding model emits its own flat
copy of the embedded fields, and a client generator can't recover the shared
base type.

`DefaultAllOfForEmbeds` changes that. With the option on, a **plain** embed (one
with no explicit name and no `swagger:allOf` tag) is rendered as an `allOf`
member — exactly as if it carried
[`swagger:allOf`]({{% relref "/tutorials/polymorphic-models" %}}) — so the
composition relationship survives in the spec. It is opt-in and defaults to off;
with it off, output is byte-identical to before.

## What composes

This model embeds a `swagger:model` type (`Base`), a non-model type (`Mixin`),
and adds an own field:

{{< code file="shaping/embedallof/embedallof.go" lang="go" region="base" >}}

{{< code file="shaping/embedallof/embedallof.go" lang="go" region="plain" >}}

Scanned with the flag off the embedded properties inline flat; on, the embed
becomes an `allOf` composition:

{{< compare left="shaping/embedallof/testdata/plainembed_off.json" leftlabel="Default — inlined"
            right="shaping/embedallof/testdata/plainembed_on.json" rightlabel="DefaultAllOfForEmbeds — composed" >}}

Reading the composed pane, each embed takes the path its kind dictates:

- **A model embed becomes a `$ref` member.** `Base` is a `swagger:model`, so it
  has its own definition and composes as `{$ref: "#/definitions/Base"}` — no
  copy of `id` / `name`.
- **A non-model embed becomes an inline member.** `Mixin` carries no
  `swagger:model`, so it has no definition to point at; its `note` property
  rides an inline `allOf` member instead.
- **The embedding struct's own fields move to a sibling member.** `color` is no
  longer a top-level property — it lands in its own `allOf` arm alongside the
  composed embeds.

## What's left alone

The flag only changes the *untagged, unnamed* embed — every other embed shape is
unaffected:

{{< code file="shaping/embedallof/embedallof.go" lang="go" region="edges" >}}

- **Pointer embeds** are peeled first, so `*Base` composes to the same
  `$ref` member as a value embed.
- **A json-named embed is not a promotion.** Giving the embed a json tag
  (`Base \`json:"base"\``) makes it a single nested property named `base`, on or
  off — Go doesn't promote a named embed (go-swagger#2038).
- **An explicit `swagger:allOf` embed already composes**, so the flag is a no-op
  for it; it only makes `allOf` the *default* for untagged embeds.
- **Interface embeds** compose via `allOf` regardless of this flag.

{{% notice style="note" %}}
`DefaultAllOfForEmbeds` is the global default-on switch for the same shape
`swagger:allOf` produces per-embed. Reach for the annotation when only some
embeds should compose; reach for the option when composition is your house style
for every plain embed.
{{% /notice %}}

## What's next

- [Polymorphic models]({{% relref "/tutorials/polymorphic-models" %}}) — the
  `swagger:allOf` annotation and discriminator hints this option generalises.
- [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}) —
  how a description and validations render on an `allOf` member.
- [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}) —
  the definition names the composed `$ref`s point at.
