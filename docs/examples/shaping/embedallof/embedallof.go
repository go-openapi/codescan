// SPDX-License-Identifier: Apache-2.0

// Package embedallof is the test-backed witness for the "Composing embeds with
// allOf" how-to. It is scanned with DefaultAllOfForEmbeds off (an embedded
// struct's properties inline into the embedding schema) and on (a plain embed
// becomes an allOf member — a $ref when the embedded type is a model, an inline
// member otherwise — with the embedding struct's own fields in a sibling
// member). A json-named embed and an explicit swagger:allOf embed are
// unaffected by the flag.
package embedallof

// snippet:base

// Base is a reusable base model.
//
// swagger:model Base
type Base struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Mixin is a non-model embedded type (no swagger:model), reachable only through
// embedding. Under the flag it composes as an inline allOf member, since it has
// no definition of its own to $ref.
type Mixin struct {
	Note string `json:"note"`
}

// endsnippet:base

// snippet:plain

// PlainEmbed embeds a model and a non-model plainly, plus an own field.
//
// swagger:model PlainEmbed
type PlainEmbed struct {
	Base
	Mixin

	Color string `json:"color"`
}

// endsnippet:plain

// snippet:edges

// PointerEmbed embeds a model through a pointer; the pointer is peeled and
// takes the same $ref path as a value embed.
//
// swagger:model PointerEmbed
type PointerEmbed struct {
	*Base

	Tag string `json:"tag"`
}

// NamedEmbed embeds Base under an explicit json name, so Go does not promote
// it: it stays a single nested property, identical on or off (go-swagger#2038).
//
// swagger:model NamedEmbed
type NamedEmbed struct {
	Base `json:"base"`

	Extra string `json:"extra"`
}

// TaggedEmbed already composes Base via an explicit swagger:allOf tag, so the
// flag does not change its shape.
//
// swagger:model TaggedEmbed
type TaggedEmbed struct {
	// swagger:allOf
	Base

	Field string `json:"field"`
}

// endsnippet:edges
