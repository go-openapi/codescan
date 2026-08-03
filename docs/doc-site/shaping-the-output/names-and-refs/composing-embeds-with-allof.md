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

## Composition needs a marshaller you write

An `allOf` says the JSON document satisfies every member at once — one flat object carrying all
their properties. Go's **default** marshaller only produces that shape by coincidence, and the
coincidence holds for exactly one case: a plain struct embed with no marshaller of its own, whose
fields Go promotes.

Step outside that case and the default rendering stops matching the spec:

- a member that is **not a struct** — a map, a slice, a named basic — promotes nothing, so Go emits
  it as one key named after the type instead of merging it;
- a member with **its own `MarshalJSON`/`MarshalText`** is promoted into your type's method set, and
  `json.Marshal` then consults it *before* reading any field — rendering the whole struct as whatever
  that method returns.

This is why go-swagger's generated models never rely on the default. A model with `allOf` embeds its
members **and** carries a hand-written pair that flattens them, reading every member from the same
raw document:

```go
// swagger:model WithAllOf
type WithAllOf struct {
	Notable                             // an allOf member

	AO1 map[string]int32 `json:"-"`     // a map member — json:"-" keeps the default out of the way

	WithAllOfAO2P2                      // another member

	Body  string `json:"body,omitempty"`   // the model's own fields
	Title string `json:"title,omitempty"`
}

// UnmarshalJSON reads every member from the SAME document — that is what allOf means.
func (m *WithAllOf) UnmarshalJSON(raw []byte) error {
	var aO0 Notable
	if err := jsonutils.ReadJSON(raw, &aO0); err != nil {
		return err
	}
	m.Notable = aO0

	var aO1 map[string]int32
	if err := jsonutils.ReadJSON(raw, &aO1); err != nil {
		return err
	}
	m.AO1 = aO1

	// … one block per member, then the model's own fields
}
```

{{% notice style="warning" %}}
If you hand-write the Go types that codescan scans, `swagger:allOf` describes your **intent**; it
does not make `encoding/json` produce that document. Write the marshaller, or generate the model
from the spec and let go-swagger write it for you. codescan reads declarations — it cannot tell
whether the marshalling you need exists, so it will not warn you.
{{% /notice %}}

Because of this, codescan reads an embed as *composition* and never as an instruction about the
default marshaller. In particular, a promoted `MarshalText`/`MarshalJSON` on an embedded type is
**not** treated as a claim that the whole model is a scalar — see
[Forcing a conformant format]({{% relref "forcing-a-format" %}}) if you want a type rendered as
one.

## Annotate the embedded type, not the embed

A classifier annotation in an **embedded field's** doc comment does nothing.
`swagger:strfmt` and `swagger:type` written there are ignored — codescan reports
them under `scan.ineffective-annotation` rather than dropping them quietly:

```go
type Wrong struct {
	// swagger:strfmt uuid   ← ignored, and warned about
	Token
}
```

An embed contributes the shape of the type it embeds, and what that shape is
comes from that type's own declaration. Put the annotation there and every embed
of it composes the same way:

```go
// swagger:strfmt uuid
type Token [16]byte
```

The catch is that both annotations *are* honoured on an ordinary field, so the
same line means something one field down and nothing on an embed. Only
`swagger:allOf`, [`swagger:omit`]({{% relref "/maintainers/annotations/swagger-omit" %}}),
`swagger:name`, `swagger:ignore` and a `required:` inheritance hint act on an
embed itself — everything else describes the embedded type and belongs with it.

## When an override cannot be composed

Composition has one limit worth knowing. Inlining an embed **resolves** an
override — a field the enclosing struct re-declares wins, exactly as Go's depth
rule decides it. `allOf` instead **accumulates**: members conjoin, and a
conjunction can only narrow, never replace. So a re-declaration that *replaces*
is not expressible as composition:

- re-declaring a promoted field to decorate it (add `readOnly`, a description, a
  validation) leaves the property in **both** members — valid, but a generator
  walking the members sees it twice;
- re-declaring it with a **different type** yields `{type: integer}` **and**
  `{type: string}` for one property — a schema nothing can satisfy.

codescan does not guess which declaration you meant: many Go types can be
written whose composition has no faithful schema, and inventing one would be
deciding your intent. Resolve it yourself with
[`swagger:omit`]({{% relref "/maintainers/annotations/swagger-omit" %}}) on the
embed, which drops the promoted twin so only your re-declaration survives:

```go
type Decorated struct {
	// swagger:omit ID
	Base

	// ID is assigned by the server.
	//
	// read only: true
	ID int64
}
```

## What's next

- [`swagger:omit`]({{% relref "/maintainers/annotations/swagger-omit" %}}) — drop
  what an embed promotes but the API should not carry.
- [Polymorphic models]({{% relref "/tutorials/polymorphic-models" %}}) — the
  `swagger:allOf` annotation and discriminator hints this option generalises.
- [Descriptions beside a `$ref`]({{% relref "descriptions-beside-a-ref" %}}) —
  how a description and validations render on an `allOf` member.
- [Resolving `$ref` name conflicts]({{% relref "resolving-name-conflicts" %}}) —
  the definition names the composed `$ref`s point at.
