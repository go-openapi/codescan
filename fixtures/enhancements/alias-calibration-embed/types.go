// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_calibration_embed is the cycle-3 calibration fixture
// for the W3 alias workshop. It tests the composition axis: how the
// schema builder shapes a struct that EMBEDS another type, in four
// permutations of how the embedded type is referenced:
//
//   1. Named struct, direct       (EmbedsDirectStruct)
//   2. Alias of struct            (EmbedsAlias)
//   3. Pointer to named struct    (EmbedsPointer)
//   4. Named interface            (EmbedsInterface)
//
// In Go, (1), (2), and (3) refer to the SAME underlying type
// (`Base`); `BaseAlias = Base` is a transparent rename, `*Base` is
// just a pointer-to-Base. The type system cannot distinguish them.
// Yet the schema builder today produces THREE different output
// shapes for the same composition intent — the central Q8 question.
//
// The B2 probe from fix-quirks revealed:
//   - EmbedsDirectStruct → FLAT inline (id, name, extra)
//   - EmbedsAlias        → allOf: [{$ref: BaseAlias}, {extra inline}]
//   - EmbedsPointer      → FLAT inline (pointer peeled → named-direct)
//   - EmbedsInterface    → FLAT inline (interface methods promoted)
//
// So the OUTLIER is alias-embed, not interface-embed (the Q8
// original framing was wrong). Cycle 3 calibrates the workshop's
// vocabulary on whether this asymmetry should hold, collapse, or
// invert.
//
// Envelope also includes a field-site comparison (Base vs
// BaseAlias as field types) so the field-reach behaviour is on
// the same canvas as the embed-reach behaviour.
//
// 5 decls × 3 modes = 15 base cells, plus indirect impacts on Base /
// BaseAlias / Methods which may surface as standalone definitions.
//
// See `.claude/plans/workshops/alias-matrix.md` §5 and
// `.claude/plans/workshops/alias-ledger.md` cycle 3.
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

// EmbedsDirectStruct embeds Base directly. The B2 probe shape:
// FLAT — `{type: object, properties: {id, name, extra}}` with no
// allOf composition. Q8 baseline.
//
// swagger:model EmbedsDirectStruct
type EmbedsDirectStruct struct {
	Base

	// Extra is a field unique to the outer struct.
	Extra string `json:"extra"`
}

// EmbedsAlias embeds BaseAlias. The B2 probe shape: `allOf:
// [{$ref: BaseAlias}, {properties: {extra}}]`. Q8 OUTLIER — same
// Go type as EmbedsDirectStruct produces a DIFFERENT spec shape.
//
// swagger:model EmbedsAlias
type EmbedsAlias struct {
	BaseAlias

	Extra string `json:"extra"`
}

// EmbedsPointer embeds *Base. The B2 probe shape: FLAT (pointer is
// peeled, then takes the named-direct path). Consistent with
// EmbedsDirectStruct.
//
// swagger:model EmbedsPointer
type EmbedsPointer struct {
	*Base

	Extra string `json:"extra"`
}

// EmbedsInterface embeds the named Methods interface. The B2 probe
// shape: FLAT — method properties promoted into the outer schema
// alongside Tag. Consistent with EmbedsDirectStruct (named embed
// of any kind goes through buildNamedEmbedded which inlines).
//
// swagger:model EmbedsInterface
type EmbedsInterface struct {
	Methods

	Tag string `json:"tag"`
}

// Envelope compares Base and BaseAlias at the FIELD-reach site
// (not embed). Cycles 1-2 showed annotated aliases produce a
// $ref to the def at field sites; we want to see whether the
// alias-vs-named asymmetry persists at the field site or only at
// the embed site.
//
// swagger:model Envelope
type Envelope struct {
	// Direct field of type Base.
	Direct Base `json:"direct"`

	// ViaAlias field of type BaseAlias — same underlying as Direct.
	ViaAlias BaseAlias `json:"viaAlias"`
}

// EmbedsAliasOptIn validates the bidirectional Q-D contract: when
// the user EXPLICITLY annotates an aliased embed with
// `swagger:allOf`, the resulting spec MUST use allOf composition
// with a $ref to the embedded type. This is the shape that was
// silently produced (without annotation) before the Q-D fix; it
// remains available as an explicit opt-in.
//
// Combined with EmbedsAlias above, this fixture asserts:
//
//   - Without `swagger:allOf` → flat inline (EmbedsAlias)
//   - With    `swagger:allOf` → allOf with $ref (EmbedsAliasOptIn)
//
// The annotation is the SOLE gate; no other input flips composition.
//
// swagger:model EmbedsAliasOptIn
type EmbedsAliasOptIn struct {
	// swagger:allOf
	BaseAlias

	// Extra is a field unique to the outer struct.
	Extra string `json:"extra"`
}

// EmbedsDirectStructOptIn mirrors EmbedsAliasOptIn but with a
// direct named-struct embed instead of an alias embed — checks
// that the explicit annotation works on non-aliased embeds too.
// Pre-Q-D this case ALREADY worked (named-direct embed with
// swagger:allOf → allOf with $ref to Base) — the annotation path
// was never broken; only the *implicit* aliased-embed promotion
// was wrong. Including this here triangulates: same annotation,
// same shape, regardless of alias or direct.
//
// swagger:model EmbedsDirectStructOptIn
type EmbedsDirectStructOptIn struct {
	// swagger:allOf
	Base

	Extra string `json:"extra"`
}
