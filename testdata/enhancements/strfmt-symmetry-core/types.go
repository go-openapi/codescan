// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_symmetry_core exercises the strfmt annotation across every dispatch
// site reachable through `buildFromType` — the busiest of `buildAlias`'s four
// callers (`schema.go:351`).
//
// Every type here is one half of a PAIR: a named declaration and an alias
// declaration over the SAME right-hand side, carrying the SAME annotation, used
// at the SAME sites. Only the `=` differs. The named half is the control — it
// defines what the cell's correct output is — so the ledger test compares the
// two halves against each other and never hand-writes an expectation.
//
// # RHS kinds
//
// Each kind is here because it selects a DIFFERENT classifier on the named side,
// so each is a distinct thing the alias side would have to learn:
//
//   - basic  → classifierNamedBasic
//   - struct → classifierNamedStructStrfmt (strfmt on a struct replaces the whole type)
//   - slice  → classifierNamedArrayLike; the "byte" format is a whole-schema
//     special, NOT an items one
//   - array  → classifierNamedArrayLike; the `bsonobjectid` special fires for
//     arrays only, never slices
//   - chain  → inheritedStrfmt; the annotation sits one declaration to the RIGHT
//
// # Cells exercised
//
//	decl        × {basic, struct, slice, array, chain} × {named, alias}
//	field       × {basic, struct, slice, array, chain} × {named, alias}
//	pointer     × {basic, struct}                      × {named, alias}
//	slice elem  × {basic, struct}                      × {named, alias}
//	map value   × {basic, struct}                      × {named, alias}
//
// Pointer / slice-elem / map-value are restricted to {basic, struct} on purpose:
// all three recurse straight back into `buildFromType`, so they re-enter the same
// dispatch the plain field cell already covers. They are here to prove that, not
// to multiply rows.
//
// `EnvelopeModeled` adds the model-annotation axis: the same pairs declared as
// first-class models, which gates whether `buildAlias` dissolves at the use site
// (`schema.go:426`).
//
// See [§aliases](../../../internal/builders/schema/README.md#aliases).
package strfmt_symmetry_core

// PlainTarget is an unannotated struct used as the RHS of the struct-kind pair.
// It carries no format annotation of its own, so any format on the pair's members
// can only come from the pair's own declaration.
type PlainTarget struct {
	// Left is a plain field, present so the struct has observable content when
	// the format is NOT applied.
	Left string `json:"left"`

	// Right is a plain field.
	Right int32 `json:"right"`
}

// BaseFormatted is the right-hand end of the chain pair: the declaration that
// actually carries the annotation. `StrfmtChainNamed` / `StrfmtChainAlias` are
// declared OVER it and carry none of their own.
//
// swagger:strfmt ssn
type BaseFormatted string

// StrfmtBasicNamed is a named type over a primitive.
//
// swagger:strfmt isbn
type StrfmtBasicNamed string

// StrfmtBasicAlias is an alias over the same primitive, same annotation.
//
// swagger:strfmt isbn
type StrfmtBasicAlias = string

// StrfmtStructNamed is a named type over a struct.
//
// swagger:strfmt duration
type StrfmtStructNamed PlainTarget

// StrfmtStructAlias is an alias over the same struct, same annotation.
//
// swagger:strfmt duration
type StrfmtStructAlias = PlainTarget

// StrfmtSliceNamed is a named type over a byte slice. `byte` is the whole-schema
// special in classifierNamedArrayLike — it must NOT land on items.
//
// swagger:strfmt byte
type StrfmtSliceNamed []byte

// StrfmtSliceAlias is an alias over the same byte slice, same annotation.
//
// swagger:strfmt byte
type StrfmtSliceAlias = []byte

// StrfmtArrayNamed is a named type over a fixed byte array. `bsonobjectid` is the
// array-only special — the slice half of classifierNamedArrayLike does not have it.
//
// swagger:strfmt bsonobjectid
type StrfmtArrayNamed [12]byte

// StrfmtArrayAlias is an alias over the same fixed array, same annotation.
//
// swagger:strfmt bsonobjectid
type StrfmtArrayAlias = [12]byte

// StrfmtChainNamed is a named type declared over an ALREADY-annotated named type.
// It carries no annotation itself: the format must be inherited from
// BaseFormatted, one declaration to the right (inheritedStrfmt).
type StrfmtChainNamed BaseFormatted

// StrfmtChainAlias is an alias over the same annotated named type, also carrying
// no annotation of its own.
type StrfmtChainAlias = BaseFormatted

// Envelope reaches every pair from a use site. Field names are the lower-camel
// cell ID, so a golden diff names its own cell.
//
// swagger:model Envelope
type Envelope struct {
	// --- field × {basic, struct, slice, array, chain} ---

	// FieldBasicNamed is the basic pair, named half, in plain field position.
	FieldBasicNamed StrfmtBasicNamed `json:"fieldBasicNamed"`

	// FieldBasicAlias is the basic pair, alias half, in plain field position.
	FieldBasicAlias StrfmtBasicAlias `json:"fieldBasicAlias"`

	// FieldStructNamed is the struct pair, named half.
	FieldStructNamed StrfmtStructNamed `json:"fieldStructNamed"`

	// FieldStructAlias is the struct pair, alias half.
	FieldStructAlias StrfmtStructAlias `json:"fieldStructAlias"`

	// FieldSliceNamed is the slice pair, named half.
	FieldSliceNamed StrfmtSliceNamed `json:"fieldSliceNamed"`

	// FieldSliceAlias is the slice pair, alias half.
	FieldSliceAlias StrfmtSliceAlias `json:"fieldSliceAlias"`

	// FieldArrayNamed is the array pair, named half.
	FieldArrayNamed StrfmtArrayNamed `json:"fieldArrayNamed"`

	// FieldArrayAlias is the array pair, alias half.
	FieldArrayAlias StrfmtArrayAlias `json:"fieldArrayAlias"`

	// FieldChainNamed is the chain pair, named half — format inherited from BaseFormatted.
	FieldChainNamed StrfmtChainNamed `json:"fieldChainNamed"`

	// FieldChainAlias is the chain pair, alias half.
	FieldChainAlias StrfmtChainAlias `json:"fieldChainAlias"`

	// --- pointer × {basic, struct} ---

	// PointerBasicNamed reaches the basic pair's named half through a pointer.
	PointerBasicNamed *StrfmtBasicNamed `json:"pointerBasicNamed"`

	// PointerBasicAlias reaches the basic pair's alias half through a pointer.
	PointerBasicAlias *StrfmtBasicAlias `json:"pointerBasicAlias"`

	// PointerStructNamed reaches the struct pair's named half through a pointer.
	PointerStructNamed *StrfmtStructNamed `json:"pointerStructNamed"`

	// PointerStructAlias reaches the struct pair's alias half through a pointer.
	PointerStructAlias *StrfmtStructAlias `json:"pointerStructAlias"`

	// --- slice element × {basic, struct} ---

	// SliceElemBasicNamed reaches the basic pair's named half as a slice element.
	SliceElemBasicNamed []StrfmtBasicNamed `json:"sliceElemBasicNamed"`

	// SliceElemBasicAlias reaches the basic pair's alias half as a slice element.
	SliceElemBasicAlias []StrfmtBasicAlias `json:"sliceElemBasicAlias"`

	// SliceElemStructNamed reaches the struct pair's named half as a slice element.
	SliceElemStructNamed []StrfmtStructNamed `json:"sliceElemStructNamed"`

	// SliceElemStructAlias reaches the struct pair's alias half as a slice element.
	SliceElemStructAlias []StrfmtStructAlias `json:"sliceElemStructAlias"`

	// --- map value × {basic, struct} ---

	// MapValueBasicNamed reaches the basic pair's named half as a map value.
	MapValueBasicNamed map[string]StrfmtBasicNamed `json:"mapValueBasicNamed"`

	// MapValueBasicAlias reaches the basic pair's alias half as a map value.
	MapValueBasicAlias map[string]StrfmtBasicAlias `json:"mapValueBasicAlias"`

	// MapValueStructNamed reaches the struct pair's named half as a map value.
	MapValueStructNamed map[string]StrfmtStructNamed `json:"mapValueStructNamed"`

	// MapValueStructAlias reaches the struct pair's alias half as a map value.
	MapValueStructAlias map[string]StrfmtStructAlias `json:"mapValueStructAlias"`
}

// ModeledBasicNamed is the basic pair's named half WITH a model annotation.
//
// swagger:model ModeledBasicNamed
// swagger:strfmt isbn
type ModeledBasicNamed string

// ModeledBasicAlias is the basic pair's alias half WITH a model annotation — the
// annotation that stops buildAlias dissolving at the use site (schema.go:426).
//
// swagger:model ModeledBasicAlias
// swagger:strfmt isbn
type ModeledBasicAlias = string

// ModeledStructNamed is the struct pair's named half WITH a model annotation.
//
// swagger:model ModeledStructNamed
// swagger:strfmt duration
type ModeledStructNamed PlainTarget

// ModeledStructAlias is the struct pair's alias half WITH a model annotation.
//
// swagger:model ModeledStructAlias
// swagger:strfmt duration
type ModeledStructAlias = PlainTarget

// EnvelopeModeled reaches the model-annotated pairs from a use site.
//
// swagger:model EnvelopeModeled
type EnvelopeModeled struct {
	// ModeledBasicNamed is the annotated basic pair, named half.
	ModeledBasicNamed ModeledBasicNamed `json:"modeledBasicNamed"`

	// ModeledBasicAlias is the annotated basic pair, alias half.
	ModeledBasicAlias ModeledBasicAlias `json:"modeledBasicAlias"`

	// ModeledStructNamed is the annotated struct pair, named half.
	ModeledStructNamed ModeledStructNamed `json:"modeledStructNamed"`

	// ModeledStructAlias is the annotated struct pair, alias half.
	ModeledStructAlias ModeledStructAlias `json:"modeledStructAlias"`
}
