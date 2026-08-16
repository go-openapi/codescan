// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package default_allof_embeds exercises Options.DefaultAllOfForEmbeds — the
// opt-in that renders a plain struct embed as allOf composition instead of
// inlining the embedded type's properties.
//
// Each embedding model below covers one interaction:
//
//   - PlainEmbed   — a model embed and a non-model embed, both untagged, plus
//     an own field. ON: model embed becomes an allOf $ref member, the non-model
//     embed an inline allOf member, and the own field a sibling member. OFF:
//     every embedded field is inlined flat.
//   - PointerEmbed — a plain *Base embed; the pointer is peeled and takes the
//     same path as PlainEmbed's model embed.
//   - NamedEmbed   — an embed carrying a json tag name. It is a single nested
//     property (Go does not promote it), so it is identical ON and OFF
//     (go-swagger#2038).
//   - TaggedEmbed  — an explicit allOf-tagged embed. Already composition, so it
//     is identical ON and OFF (the flag only changes the untagged default).
//
// See internal/builders/schema/README.md §allof.
package default_allof_embeds

// Base is a reusable base model.
//
// swagger:model Base
type Base struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Mixin is a non-model embedded type (no swagger:model), reachable only through
// embedding. Under the flag it composes as an inline allOf member rather than a
// $ref, since it has no definition of its own.
type Mixin struct {
	Note string `json:"note"`
}

// PlainEmbed embeds a model and a non-model plainly, plus an own field.
//
// swagger:model PlainEmbed
type PlainEmbed struct {
	Base
	Mixin
	Color string `json:"color"`
}

// PointerEmbed embeds a model through a pointer.
//
// swagger:model PointerEmbed
type PointerEmbed struct {
	*Base
	Tag string `json:"tag"`
}

// NamedEmbed embeds Base under an explicit json name; it nests as a single
// property and is unaffected by the flag.
//
// swagger:model NamedEmbed
type NamedEmbed struct {
	Base  `json:"base"`
	Extra string `json:"extra"`
}

// TaggedEmbed composes Base through an explicit allOf tag; the flag does not
// change its shape.
//
// swagger:model TaggedEmbed
type TaggedEmbed struct {
	// swagger:allOf
	Base
	Field string `json:"field"`
}
