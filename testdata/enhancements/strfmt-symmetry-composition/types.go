// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_symmetry_composition exercises the strfmt annotation at the two
// COMPOSITION dispatch sites — the ones that do not go through `buildFromType`:
//
//   - plain struct embed → `buildEmbedded` (`embedded.go:40`), which unaliases and
//     recurses into `buildNamedEmbedded`
//   - allOf member       → `buildAllOf` (`allof.go:172`), whose named arm
//     `buildNamedAllOf` runs `classifierAliasTargetStrfmt` while its alias arm
//     drops straight into `buildAlias`
//
// Each composing type is one half of a PAIR: same member type, same annotation,
// same composition, differing only by the member's `=`. The pair is compared at
// DEFINITION level, because a composition has no enclosing property to inspect.
//
// # The third composition site carries no cell, deliberately
//
// `processEmbeddedType` (`embedded.go:155`) is the interface-side allOf walk and
// is the fourth caller of `buildAlias`. Go interfaces can only embed interfaces,
// so the only alias reachable there is an alias-to-interface — and no classifier
// consumes a format on an interface underlying on EITHER side (the named arm goes
// to `resolveRefOrErr`). There is no meaningful strfmt cell to write; the site is
// out of scope for this matrix rather than untested.
//
// See [§aliases](../../../internal/builders/schema/README.md#aliases) and
// [§allof](../../../internal/builders/schema/README.md#allof).
package strfmt_symmetry_composition

// PlainTarget is the unannotated struct behind the struct-kind pair.
type PlainTarget struct {
	// Left is a plain field.
	Left string `json:"left"`

	// Right is a plain field.
	Right int32 `json:"right"`
}

// FmtBasicNamed is a named type over a primitive, carrying a format.
//
// swagger:strfmt isbn
type FmtBasicNamed string

// FmtBasicAlias is an alias over the same primitive, same format.
//
// swagger:strfmt isbn
type FmtBasicAlias = string

// FmtStructNamed is a named type over a struct, carrying a format.
//
// swagger:strfmt duration
type FmtStructNamed PlainTarget

// FmtStructAlias is an alias over the same struct, same format.
//
// swagger:strfmt duration
type FmtStructAlias = PlainTarget

// --- plain struct embed: buildEmbedded ---

// EmbedBasicNamed plainly embeds the basic pair's named half.
//
// swagger:model EmbedBasicNamed
type EmbedBasicNamed struct {
	FmtBasicNamed

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// EmbedBasicAlias plainly embeds the basic pair's alias half.
//
// swagger:model EmbedBasicAlias
type EmbedBasicAlias struct {
	FmtBasicAlias

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// EmbedStructNamed plainly embeds the struct pair's named half.
//
// swagger:model EmbedStructNamed
type EmbedStructNamed struct {
	FmtStructNamed

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// EmbedStructAlias plainly embeds the struct pair's alias half.
//
// swagger:model EmbedStructAlias
type EmbedStructAlias struct {
	FmtStructAlias

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// --- allOf member: buildAllOf ---

// AllOfBasicNamed composes the basic pair's named half as an allOf member.
//
// swagger:model AllOfBasicNamed
type AllOfBasicNamed struct {
	// swagger:allOf
	FmtBasicNamed

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// AllOfBasicAlias composes the basic pair's alias half as an allOf member.
//
// swagger:model AllOfBasicAlias
type AllOfBasicAlias struct {
	// swagger:allOf
	FmtBasicAlias

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// AllOfStructNamed composes the struct pair's named half as an allOf member.
//
// swagger:model AllOfStructNamed
type AllOfStructNamed struct {
	// swagger:allOf
	FmtStructNamed

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// AllOfStructAlias composes the struct pair's alias half as an allOf member.
//
// swagger:model AllOfStructAlias
type AllOfStructAlias struct {
	// swagger:allOf
	FmtStructAlias

	// Note is the composing struct's own field.
	Note string `json:"note"`
}
