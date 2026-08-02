// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package type_override_symmetry witnesses `swagger:type` on a NAMED declaration
// against the same annotation on an ALIAS one, at every dispatch site.
//
// Sibling of `strfmt-symmetry-core`, for the other half of Q32. `swagger:type`
// has a wider surface than `swagger:strfmt` — a scalar / Go-builtin / OAS-2 name,
// `[]T` array prefixes, the `inline` keyword, and a reference to another scanned
// type — so each form gets its own pair rather than assuming they share a path.
//
// Every cell is a PAIR differing only by the `=`. The named half is the control:
// it defines the cell's correct output, so nothing here hard-codes an expectation.
//
// Sites covered: declaration, struct field, pointer, slice element, map value,
// allOf member, and the SimpleSchema locations (non-body parameter, response
// header). The last group is deliberate — the parameters and responses builders
// have their own dispatch, and a fix verified only on the schema builder has
// already once looked complete when it was not.
package type_override_symmetry

// PlainTarget is a struct used as the target of a type-name reference override.
type PlainTarget struct {
	// Left is a plain field.
	Left string `json:"left"`

	// Right is a plain field.
	Right int32 `json:"right"`
}

// --- scalar override: an int declared to be a string ---

// ScalarNamed is a named int overridden to a string.
//
// swagger:type string
type ScalarNamed int

// ScalarAlias is an alias to the same int, same override.
//
// swagger:type string
type ScalarAlias = int

// --- array override: `[]T` prefixes ---

// ArrayNamed is a named string overridden to an array of strings.
//
// swagger:type []string
type ArrayNamed string

// ArrayAlias is an alias to the same string, same override.
//
// swagger:type []string
type ArrayAlias = string

// --- type-name reference: inline another scanned type in place ---

// RefNamed is a named int overridden to PlainTarget's shape.
//
// swagger:type PlainTarget
type RefNamed int

// RefAlias is an alias to the same int, same override.
//
// swagger:type PlainTarget
type RefAlias = int

// --- swagger:type with a co-present swagger:strfmt (type wins, format advisory) ---

// FormattedNamed is a named int overridden to a string carrying a format.
//
// swagger:type string
// swagger:strfmt uuid
type FormattedNamed int

// FormattedAlias is the alias half of the same pair.
//
// swagger:type string
// swagger:strfmt uuid
type FormattedAlias = int

// Envelope reaches every pair from the buildFromType sites.
//
// swagger:model Envelope
type Envelope struct {
	// FieldScalarNamed is the scalar pair, named half.
	FieldScalarNamed ScalarNamed `json:"fieldScalarNamed"`

	// FieldScalarAlias is the scalar pair, alias half.
	FieldScalarAlias ScalarAlias `json:"fieldScalarAlias"`

	// FieldArrayNamed is the array pair, named half.
	FieldArrayNamed ArrayNamed `json:"fieldArrayNamed"`

	// FieldArrayAlias is the array pair, alias half.
	FieldArrayAlias ArrayAlias `json:"fieldArrayAlias"`

	// FieldRefNamed is the type-reference pair, named half.
	FieldRefNamed RefNamed `json:"fieldRefNamed"`

	// FieldRefAlias is the type-reference pair, alias half.
	FieldRefAlias RefAlias `json:"fieldRefAlias"`

	// FieldFormattedNamed is the type+strfmt pair, named half.
	FieldFormattedNamed FormattedNamed `json:"fieldFormattedNamed"`

	// FieldFormattedAlias is the type+strfmt pair, alias half.
	FieldFormattedAlias FormattedAlias `json:"fieldFormattedAlias"`

	// PointerScalarNamed reaches the scalar pair's named half through a pointer.
	PointerScalarNamed *ScalarNamed `json:"pointerScalarNamed"`

	// PointerScalarAlias reaches the scalar pair's alias half through a pointer.
	PointerScalarAlias *ScalarAlias `json:"pointerScalarAlias"`

	// SliceElemScalarNamed reaches the scalar pair's named half as a slice element.
	SliceElemScalarNamed []ScalarNamed `json:"sliceElemScalarNamed"`

	// SliceElemScalarAlias reaches the scalar pair's alias half as a slice element.
	SliceElemScalarAlias []ScalarAlias `json:"sliceElemScalarAlias"`

	// MapValueScalarNamed reaches the scalar pair's named half as a map value.
	MapValueScalarNamed map[string]ScalarNamed `json:"mapValueScalarNamed"`

	// MapValueScalarAlias reaches the scalar pair's alias half as a map value.
	MapValueScalarAlias map[string]ScalarAlias `json:"mapValueScalarAlias"`
}

// --- allOf members ---

// AllOfScalarNamed composes the scalar pair's named half as an allOf member.
//
// swagger:model AllOfScalarNamed
type AllOfScalarNamed struct {
	// swagger:allOf
	ScalarNamed

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// AllOfScalarAlias composes the scalar pair's alias half as an allOf member.
//
// swagger:model AllOfScalarAlias
type AllOfScalarAlias struct {
	// swagger:allOf
	ScalarAlias

	// Note is the composing struct's own field.
	Note string `json:"note"`
}

// --- SimpleSchema sites ---

// TypeParams carries the pairs in non-body parameter positions.
//
// swagger:parameters typeOverrideOp
type TypeParams struct {
	// QueryScalarNamed is the scalar pair's named half in query position.
	//
	// in: query
	QueryScalarNamed ScalarNamed `json:"queryScalarNamed"`

	// QueryScalarAlias is the scalar pair's alias half in query position.
	//
	// in: query
	QueryScalarAlias ScalarAlias `json:"queryScalarAlias"`
}

// TypeResponse carries the pairs on response headers.
//
// swagger:response typeOverrideResponse
type TypeResponse struct {
	// HeaderScalarNamed is the scalar pair's named half in header position.
	//
	// in: header
	HeaderScalarNamed ScalarNamed `json:"X-Named"`

	// HeaderScalarAlias is the scalar pair's alias half in header position.
	//
	// in: header
	HeaderScalarAlias ScalarAlias `json:"X-Alias"`
}

// --- `swagger:type file` is a synonym for `swagger:file` ---
//
// `file` is an OAS v2 type name like any other, so the annotation that names
// types should be able to name it; `swagger:file` is the older, extraneous
// spelling and is expected to be deprecated. The two must produce identical
// output wherever `file` is legal — a formData parameter and a response body —
// and the location gate is shared, so neither spelling can leak `file` into a
// place OAS 2.0 forbids.

// FileParams pairs the two spellings in formData, plus an illegal location.
//
// swagger:parameters fileSynonymOp
type FileParams struct {
	// ViaAnnotation uses the legacy spelling.
	//
	// swagger:file
	// in: formData
	ViaAnnotation interface{} `json:"viaAnnotation"`

	// ViaType uses the preferred spelling and must match it exactly.
	//
	// swagger:type file
	// in: formData
	ViaType interface{} `json:"viaType"`

	// QueryFile is illegal: `file` is formData-only, so the override is refused
	// and the Go type stands.
	//
	// swagger:type file
	// in: query
	QueryFile string `json:"queryFile"`
}

// FileBodyAnnotation is a file-download response, legacy spelling.
//
// swagger:response fileBodyAnnotation
type FileBodyAnnotation struct {
	// Body is the download.
	//
	// swagger:file
	// in: body
	Body interface{} `json:"body"`
}

// FileBodyType is the same response via the preferred spelling.
//
// swagger:response fileBodyType
type FileBodyType struct {
	// Body is the download.
	//
	// swagger:type file
	// in: body
	Body interface{} `json:"body"`
}

// FileHandler binds the file-synonym parameters and responses.
//
// swagger:route POST /file-synonym symmetry fileSynonymOp
//
// Responses:
//
//	200: fileBodyAnnotation
//	201: fileBodyType
func FileHandler() {}

// TypeHandler binds the parameters and the response to an operation.
//
// swagger:route GET /type-override symmetry typeOverrideOp
//
// Responses:
//
//	200: typeOverrideResponse
func TypeHandler() {}
