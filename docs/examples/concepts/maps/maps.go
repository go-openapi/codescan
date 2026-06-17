// SPDX-License-Identifier: Apache-2.0

// Package maps holds the annotated declarations for the "Maps & free-form
// objects" tutorial. maps_test.go scans it and writes the golden fragments the
// guide renders, so the documentation can never drift from the scanner's real
// output.
package maps

// snippet:naturalmap

// Inventory shows a plain Go map. A map renders as an object whose values all
// share one schema, carried as additionalProperties.
//
// swagger:model
type Inventory struct {
	// Counts maps each SKU to its on-hand quantity.
	Counts map[string]int `json:"counts"`
}

// endsnippet:naturalmap

// snippet:keytypes

// Lookups shows which Go map keys survive. A key is usable when encoding/json
// can stringify it: string kinds, every integer/unsigned kind, and types
// implementing encoding.TextMarshaler. Other keys (float, bool, struct without
// TextMarshaler) are dropped with a diagnostic.
//
// swagger:model
type Lookups struct {
	// ByName is keyed by a plain string.
	ByName map[string]int `json:"byName"`

	// ByCode is keyed by an integer — JSON stringifies it, so the map is still
	// an object with additionalProperties.
	ByCode map[int]string `json:"byCode"`
}

// endsnippet:keytypes

// Thing is referenced as an additionalProperties value type.
//
// swagger:model
type Thing struct {
	Name string `json:"name"`
}

// snippet:marker

// ClosedObject forbids any key beyond its named properties: the marker false
// closes the object.
//
// swagger:model
// swagger:additionalProperties false
type ClosedObject struct {
	A string `json:"a"`
	B int    `json:"b"`
}

// OpenObject keeps its named property and also allows arbitrary extra keys.
//
// swagger:model
// swagger:additionalProperties true
type OpenObject struct {
	A string `json:"a"`
}

// TypedObject complements its named property with typed (integer) extra values.
//
// swagger:model
// swagger:additionalProperties integer
type TypedObject struct {
	A string `json:"a"`
}

// RefObject references a model as the schema of its extra values.
//
// swagger:model
// swagger:additionalProperties Thing
type RefObject struct {
	A string `json:"a"`
}

// endsnippet:marker

// snippet:fieldkeyword

// Holder decorates individual fields with the additionalProperties: keyword.
//
// swagger:model
type Holder struct {
	// OverriddenMap keeps its map shape but overrides the value schema from the
	// Go element type (string) to integer.
	//
	// additionalProperties: integer
	OverriddenMap map[string]string `json:"overriddenMap"`

	// RefMap points the map values at a model.
	//
	// additionalProperties: Thing
	RefMap map[string]string `json:"refMap"`

	// ClosedRef references a model and forbids extra keys. Because the field is
	// a $ref, the value rides an allOf sibling so the reference is preserved.
	//
	// additionalProperties: false
	ClosedRef Thing `json:"closedRef"`
}

// endsnippet:fieldkeyword

// snippet:patterntyped

// TypedPatterns maps property-name regexes to typed value schemas. Each quoted
// regex pairs with a value spec — a primitive or a model name (which becomes a
// $ref). The pattern-properties keyword is JSON-Schema, beyond the Swagger 2.0
// subset; codescan emits it ungated.
//
// swagger:model
// swagger:patternProperties "^x-": string, "^\d+$": integer, "^item-": Thing
type TypedPatterns struct {
	Known string `json:"known"`
}

// endsnippet:patterntyped
