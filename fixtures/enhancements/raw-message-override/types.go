// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package raw_message_override exercises user-classifier overrides on
// json.RawMessage. By default a json.RawMessage value emits an empty
// schema (`{}`, "any type") via recognizeRawMessage. This fixture
// captures the three scopes where a user might want to constrain that
// to a typed schema:
//
//	A. Plain field — emits `{}` (the recognizer baseline).
//	B. Named wrapper type with `swagger:type` on the decl —
//	   honoured at both field-reference sites (classifierNamedArrayLike)
//	   AND at the wrapper's own top-level definition (buildFromDecl
//	   runs classifierNamedTypeOverride before kind-dispatch, falling
//	   back to Underlying() on unknown-leaf values like "array").
//	C. Field-level `swagger:type` on a json.RawMessage field —
//	   consumed by scanFieldDoc and applied in applyFieldCarrier
//	   after buildFromType. The recognizer's empty-schema default
//	   is replaced by the user-specified type.
package raw_message_override

import "encoding/json"

// PlainContainer (case A) — bare json.RawMessage field, no overrides.
//
// swagger:model PlainContainer
type PlainContainer struct {
	// Should emit an empty schema (`{}`).
	Payload json.RawMessage `json:"payload"`
}

// AsObject (case B.1) — named wrapper of json.RawMessage with
// `swagger:type object`. The classifier overrides the recognizer
// because IsStdJSONRawMessage checks for encoding/json.RawMessage
// identity; the wrapper has its own package path and does NOT
// match. The classifier on the slice arm fires instead.
//
// swagger:model AsObject
// swagger:type object
type AsObject json.RawMessage

// AsArray (case B.2) — named wrapper with `swagger:type array`.
//
// swagger:model AsArray
// swagger:type array
type AsArray json.RawMessage

// TypedContainer (case B field reference) — references the wrappers.
//
// swagger:model TypedContainer
type TypedContainer struct {
	// Wrapped via AsObject — expect `{$ref: AsObject}` whose definition
	// resolves to `{type: object}` (the classifier wins on the decl).
	Obj AsObject `json:"obj"`

	// Wrapped via AsArray — expect `{$ref: AsArray}` whose definition
	// resolves to a typed array shape.
	Arr AsArray `json:"arr"`
}

// FieldLevelContainer (case C) — exercises field-level
// `swagger:type` on plain json.RawMessage fields. The field-level
// walker (scanFieldDoc) consumes swagger:type and applyFieldCarrier
// applies it after buildFromType, so the user override beats the
// RawMessage recognizer's empty-schema default.
//
// swagger:model FieldLevelContainer
type FieldLevelContainer struct {
	// Overridden to {type: object} via field-level swagger:type.
	//
	// swagger:type object
	Obj json.RawMessage `json:"obj"`

	// Overridden to {type: array, items: {integer, uint8}} via
	// field-level swagger:type. "array" isn't a known SwaggerSchemaForType
	// value, so the override falls back to building from Underlying()
	// (= []byte) which yields the byte-typed array shape.
	//
	// swagger:type array
	Arr json.RawMessage `json:"arr"`
}
