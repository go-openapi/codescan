// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_decl_arraylike witnesses a format annotation on a type whose
// underlying is an ARRAY or SLICE, at the declaration site.
//
// `buildFromDecl`'s underlying-kind switch has arms for struct and basic only,
// so `classifierNamedArrayLike` never fires there. Downstream cannot compensate:
// the `refModel` gate skips the inline classifiers for a model type precisely
// because the declaration is supposed to have published the override already.
//
// The existing array/slice format fixtures all use `byte` or `bsonobjectid` —
// the two whole-schema specials — where the defect is invisible because both
// land on the schema either way. Every format here is deliberately NON-special.
//
// # The two shapes are not the same question
//
//   - `[16]byte` annotated `uuid` — the format describes the WHOLE value; the
//     array is a representation detail, and an "array of uuid strings" is not
//     what the author means.
//   - `[]string` annotated `email` — the format describes each ELEMENT; an array
//     of emails is exactly what the author means.
//
// Both are here so the items-vs-whole rule can be judged on evidence rather than
// on the `byte` special that currently stands in for it.
//
// See [§aliases](../../../internal/builders/schema/README.md#aliases).
package strfmt_decl_arraylike

// IDNamedModeled is a fixed byte array that IS a uuid, published as a model.
//
// swagger:model IDNamedModeled
// swagger:strfmt uuid
type IDNamedModeled [16]byte

// IDAliasModeled is the alias half of the same pair.
//
// swagger:model IDAliasModeled
// swagger:strfmt uuid
type IDAliasModeled = [16]byte

// IDNamedPlain is the same byte array with no model annotation, so field sites
// reach the inline classifier instead of a $ref.
//
// swagger:strfmt uuid
type IDNamedPlain [16]byte

// IDAliasPlain is the alias half of the unannotated pair.
//
// swagger:strfmt uuid
type IDAliasPlain = [16]byte

// ULIDNamedModeled shows the rule generalising to a strfmt type that never had
// an entry in the old allowlist.
//
// swagger:model ULIDNamedModeled
// swagger:strfmt ulid
type ULIDNamedModeled [16]byte

// ULIDAliasModeled is the alias half of the ULID pair.
//
// swagger:model ULIDAliasModeled
// swagger:strfmt ulid
type ULIDAliasModeled = [16]byte

// RunesNamedModeled is a rune sequence — string-like for the same reason a byte
// sequence is, so the format describes the whole value.
//
// swagger:model RunesNamedModeled
// swagger:strfmt password
type RunesNamedModeled []rune

// RunesAliasModeled is the alias half of the rune pair.
//
// swagger:model RunesAliasModeled
// swagger:strfmt password
type RunesAliasModeled = []rune

// EmailsNamedModeled is a string slice whose format describes each ELEMENT.
//
// swagger:model EmailsNamedModeled
// swagger:strfmt email
type EmailsNamedModeled []string

// EmailsAliasModeled is the alias half of the element-format pair.
//
// swagger:model EmailsAliasModeled
// swagger:strfmt email
type EmailsAliasModeled = []string

// Envelope reaches the non-model pairs from a field site, where the inline
// classifier still runs.
//
// swagger:model Envelope
type Envelope struct {
	// FieldIDNamed is the whole-value format, named half, at a field site.
	FieldIDNamed IDNamedPlain `json:"fieldIdNamed"`

	// FieldIDAlias is the whole-value format, alias half, at a field site.
	FieldIDAlias IDAliasPlain `json:"fieldIdAlias"`
}
