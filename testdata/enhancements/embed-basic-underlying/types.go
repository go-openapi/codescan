// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package embed_basic_underlying exercises the embed of a named type whose UNDERLYING is
// neither a struct nor an interface.
//
// Such an embed promotes no field — there is none to promote — so Go keeps the embedded value as
// an ordinary member keyed by the TYPE NAME. `buildNamedEmbedded` had arms for struct and
// interface only, so every shape here fell to a warn-and-skip default and the member vanished
// from the schema.
//
// This is the one embed shape where the embed's own json tag is meaningful again: it names an
// ordinary property rather than steering a promotion.
//
// See [§embedded](../../../internal/builders/schema/README.md#embedded).
package embed_basic_underlying

// Count is a named type over a primitive, unannotated.
type Count int

// FmtBasic is a named type over a primitive, carrying a format, so the witness shows that the
// member is built from the embedded type — classifiers included — and not merely declared.
//
// swagger:strfmt duration
type FmtBasic int

// Codes is a named type over a slice.
type Codes []string

// Grid is a named type over an array.
type Grid [4]int32

// Token is a named type over an array that also implements encoding.TextMarshaler.
//
// Under encoding/json the promoted MarshalText makes the WHOLE embedding struct render as a bare
// string. codescan deliberately does not model that: an embed means composition, and a composed
// model round-trips through a custom marshaller rather than the default one. The member is built
// like any other instead.
type Token [16]byte

// MarshalText renders the token as text.
func (t Token) MarshalText() ([]byte, error) { return []byte("tok"), nil }

// UnmarshalText parses the token from text.
func (t *Token) UnmarshalText([]byte) error { return nil }

// BasicHost embeds a plain primitive-underlying named type.
//
// swagger:model BasicHost
type BasicHost struct {
	Count

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// FmtHost embeds a primitive-underlying named type that carries a format.
//
// swagger:model FmtHost
type FmtHost struct {
	FmtBasic

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// SliceHost embeds a slice-underlying named type.
//
// swagger:model SliceHost
type SliceHost struct {
	Codes

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// ArrayHost embeds an array-underlying named type.
//
// swagger:model ArrayHost
type ArrayHost struct {
	Grid

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// TaggedHost names the embed with a json tag, which here renames an ordinary property.
//
// swagger:model TaggedHost
type TaggedHost struct {
	Count `json:"count"`

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// OmittedHost drops the embed with `json:"-"`, exactly as it would drop a regular field.
//
// swagger:model OmittedHost
type OmittedHost struct {
	Count `json:"-"`

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// PtrHost embeds a pointer to a primitive-underlying named type.
//
// swagger:model PtrHost
type PtrHost struct {
	*Count

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// MarshalHost embeds a text-marshalable array-underlying named type.
//
// swagger:model MarshalHost
type MarshalHost struct {
	Token

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// MapHost embeds a map-underlying named type, the remaining non-struct underlying.
//
// swagger:model MapHost
type MapHost struct {
	Registry

	// Label is the embedding struct's own field.
	Label string `json:"label"`
}

// Registry is a named type over a map.
type Registry map[string]int32

// Control declares every embedded type above as an ORDINARY field.
//
// It is the calibration for what "built from the embedded type" has to mean: the member an embed
// contributes must be the same schema a plain field of that type already produces. Without it the
// expected shape would be invented here rather than derived from the builder's existing behaviour.
//
// swagger:model Control
type Control struct {
	CountField    Count    `json:"countField"`
	FmtField      FmtBasic `json:"fmtField"`
	CodesField    Codes    `json:"codesField"`
	GridField     Grid     `json:"gridField"`
	TokenField    Token    `json:"tokenField"`
	RegistryField Registry `json:"registryField"`
	PtrField      *Count   `json:"ptrField"`
}
