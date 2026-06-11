// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_calibration_embed exercises the schema builder's
// alias handling at two reach contexts:
//
//   1. Struct embedding — how does an embed produce inline-vs-allOf
//      shape across direct named, alias-of-struct, pointer, and
//      named-interface embeds?
//   2. Field site — how does an annotated alias differ from an
//      unannotated alias when referenced as a struct field?
//
// The decls below pair each direct / alias / pointer embed against
// its bidirectional companion (annotated vs unannotated alias
// variants) so the per-mode shapes are visible side by side in the
// captured goldens.
//
// See the schema builder's alias-handling contract for the rule:
// [§aliases](../../../internal/builders/schema/README.md#aliases).
package alias_calibration_embed

// Base is the canonical struct used as the embedded target in
// every variant below. Two fields so we can see the FLAT vs allOf
// distinction in the output.
//
// swagger:model Base
type Base struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// BaseAlias is a transparent rename of Base. In Go, BaseAlias and
// Base are LITERALLY the same type — `types.Unalias(BaseAlias) ==
// Base`, reflection cannot distinguish them.
type BaseAlias = Base

// Methods is a named interface with one valid property-shaped
// method, used to test embed-of-interface.
type Methods interface {
	// Describe returns a human-readable label.
	Describe() string
}

// EmbedsDirectStruct embeds Base directly.
// Expected: FLAT — `{type: object, properties: {id, name, extra}}`
// with no allOf composition.
//
// swagger:model EmbedsDirectStruct
type EmbedsDirectStruct struct {
	Base

	// Extra is a field unique to the outer struct.
	Extra string `json:"extra"`
}

// EmbedsAlias embeds BaseAlias (an unannotated alias of Base).
// Expected: FLAT — same shape as EmbedsDirectStruct, because the
// aliased embed dissolves to its named target and contributes
// inline properties (not allOf composition).
//
// swagger:model EmbedsAlias
type EmbedsAlias struct {
	BaseAlias

	Extra string `json:"extra"`
}

// EmbedsPointer embeds *Base. Expected: FLAT (pointer is peeled,
// then takes the named-direct path). Consistent with
// EmbedsDirectStruct.
//
// swagger:model EmbedsPointer
type EmbedsPointer struct {
	*Base

	Extra string `json:"extra"`
}

// EmbedsInterface embeds the named Methods interface. Expected:
// FLAT — method properties promoted into the outer schema
// alongside Tag.
//
// swagger:model EmbedsInterface
type EmbedsInterface struct {
	Methods

	Tag string `json:"tag"`
}

// Envelope compares Base and BaseAlias at the FIELD-reach site
// (not embed). BaseAlias is intentionally UNANNOTATED — at field
// sites the unannotated alias dissolves to its unaliased target,
// and the alias produces no `definitions` entry.
//
//   - direct → {$ref: Base}
//   - viaAlias → {$ref: Base}     (unannotated alias dissolves)
//
// swagger:model Envelope
type Envelope struct {
	// Direct field of type Base.
	Direct Base `json:"direct"`

	// ViaAlias field of type BaseAlias — same underlying as Direct.
	ViaAlias BaseAlias `json:"viaAlias"`
}

// EmbedsAliasOptIn pins the explicit-opt-in side of the embed
// contract: when the user annotates an aliased embed with
// `swagger:allOf`, the spec uses allOf composition with a $ref.
// The unannotated alias still dissolves to the unaliased target —
// `swagger:allOf` governs composition shape only, not identity.
//
//   - Without `swagger:allOf` → flat inline (EmbedsAlias above)
//   - With    `swagger:allOf` → allOf with $ref to Base
//
// swagger:model EmbedsAliasOptIn
type EmbedsAliasOptIn struct {
	// swagger:allOf
	BaseAlias

	// Extra is a field unique to the outer struct.
	Extra string `json:"extra"`
}

// EmbedsDirectStructOptIn mirrors EmbedsAliasOptIn but with a
// direct named-struct embed instead of an alias embed —
// triangulates that the explicit annotation works on non-aliased
// embeds too. Same annotation, same shape, regardless of alias or
// direct.
//
// swagger:model EmbedsDirectStructOptIn
type EmbedsDirectStructOptIn struct {
	// swagger:allOf
	Base

	Extra string `json:"extra"`
}

// BaseAliasModeled is a transparent rename of Base — same Go type
// as BaseAlias above — but it CARRIES `swagger:model`. The
// annotation is the user's explicit opt-in to exposing the alias
// as a first-class spec entity.
//
// At field / element / allOf-member use sites:
//
//   - BaseAlias (no annotation): aliasing is a Go implementation
//     detail; use sites dissolve to `Base`, the alias does not
//     appear in `definitions`.
//   - BaseAliasModeled (swagger:model): the alias is a first-class
//     entity; use sites keep `$ref: BaseAliasModeled` and the alias
//     gets its own definition (chain under RefAliases, full
//     structural under Expand, structural-copy under
//     TransparentAliases — modes only affect the decl shape, not
//     whether the alias surfaces).
//
// swagger:model BaseAliasModeled
type BaseAliasModeled = Base

// EmbedsAliasModeledOptIn is the bidirectional sibling of
// EmbedsAliasOptIn. Both use `swagger:allOf` on the embedded
// alias to opt into allOf composition, but they differ in whether
// the alias itself is annotated:
//
//   - EmbedsAliasOptIn        embeds BaseAlias        (UNannotated)
//     → allOf $ref dissolves to Base (alias name not exposed)
//   - EmbedsAliasModeledOptIn embeds BaseAliasModeled (annotated)
//     → allOf $ref preserves BaseAliasModeled (annotated → first-class)
//
// `swagger:allOf` governs the SHAPE (composition vs flat inline);
// `swagger:model` governs the IDENTITY (whether the alias name
// appears as a $ref target). They are orthogonal.
//
// swagger:model EmbedsAliasModeledOptIn
type EmbedsAliasModeledOptIn struct {
	// swagger:allOf
	BaseAliasModeled

	Extra string `json:"extra"`
}

// EnvelopeAnnotatedAlias is the bidirectional sibling of Envelope.
// It exposes the ANNOTATED alias `BaseAliasModeled` at a field
// site, alongside a direct `Base` field for comparison. Combined
// with Envelope (which exercises the UNANNOTATED `BaseAlias`),
// the two structs pin both halves of the field-site contract on
// one canvas:
//
//	Envelope.direct          → {$ref: Base}
//	Envelope.viaAlias        → {$ref: Base}            (unannotated dissolves)
//	EnvelopeAnnotatedAlias.direct          → {$ref: Base}
//	EnvelopeAnnotatedAlias.viaAliasModeled → {$ref: BaseAliasModeled}
//
// swagger:model EnvelopeAnnotatedAlias
type EnvelopeAnnotatedAlias struct {
	// Direct field of type Base — control for comparison.
	Direct Base `json:"direct"`

	// ViaAliasModeled — annotated alias as a field type. The
	// annotated alias keeps its identity at the use site
	// regardless of mode (Default / Ref); Transparent still
	// dissolves at use sites because Transparent supersedes
	// annotation.
	ViaAliasModeled BaseAliasModeled `json:"viaAliasModeled"`
}
