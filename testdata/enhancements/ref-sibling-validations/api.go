// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package refsiblingvalidations decorates $ref'd fields with every class of sibling keyword.
//
// A field whose type is a model becomes a $ref, and JSON Schema draft-4 says a $ref may not carry
// siblings — so anything written beside it has to ride an allOf compound instead. Which keyword
// lands where is decided by one collector, and each value shape it can receive (number, integer,
// bool, string, raw block, extension) reaches it by a different arm.
package refsiblingvalidations

// Item is the referenced model; every field below points at it.
//
// swagger:model
type Item struct {
	Name string `json:"name"`
}

// Bag carries one $ref'd field per class of sibling keyword.
//
// swagger:model
type Bag struct {
	// Ranged carries the numeric validations.
	//
	// maximum: 100
	// minimum: 1
	// multipleOf: 5
	Ranged Item `json:"ranged"`

	// Sized carries the length and item-count validations.
	//
	// minLength: 1
	// maxLength: 64
	// minItems: 2
	// maxItems: 8
	Sized Item `json:"sized"`

	// Counted carries the property-count validations.
	//
	// minProperties: 1
	// maxProperties: 4
	Counted Item `json:"counted"`

	// Flagged carries the boolean validations.
	//
	// readOnly: true
	// unique: true
	// required: true
	Flagged Item `json:"flagged"`

	// Matched carries the string-shaped validations.
	//
	// pattern: ^[a-z]+$
	// default: {"name":"fallback"}
	// example: {"name":"sample"}
	Matched Item `json:"matched"`

	// Documented carries an externalDocs block, which can only ride a compound.
	//
	// externalDocs:
	//   url: https://example.com/bag
	//   description: the bag guide
	Documented Item `json:"documented"`

	// Extended carries an author-written vendor extension, the one class that can ride beside a
	// $ref directly under EmitRefSiblings.
	//
	// Extensions:
	// ---
	// x-order: 3
	// x-internal: true
	Extended Item `json:"extended"`

	// Scalar carries the same keywords with plain scalar values, which reach the collector as
	// strings rather than as raw JSON literals.
	//
	// default: fallback
	// example: sample
	// enum: alpha,beta,gamma
	Scalar Item `json:"scalar"`

	// Keyed carries the map-shaped validations. The field-level patternProperties keyword takes a
	// single regex (the typed pair syntax belongs to the swagger:patternProperties annotation).
	//
	// patternProperties: ^x-
	// additionalProperties: string
	Keyed Item `json:"keyed"`

	// BareExt writes an extension without the Extensions block, which is not the grammar.
	//
	// x-foo: bar
	BareExt Item `json:"bareExt"`
}
