// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package provenanceshapes exercises cross-ref jsonpointer anchoring through
// the schema builder's nested shapes, so the companion provenance pointer
// accurately tracks the spec node a Go construct produced:
//
//   - inline array-of-struct  → element props anchor under …/items
//   - inline map-of-struct    → value props anchor under …/additionalProperties
//   - swagger:type []Inner    → the override inlines a named struct into array
//     items, so Inner's props anchor under …/items (the regression case for the
//     missing items-descend in resolveTypeOverride).
package provenanceshapes

// Inner is a named struct inlined as a `swagger:type []Inner` array element.
// It is deliberately NOT a swagger:model: the override inlines its underlying
// shape in place rather than emitting a $ref, so its fields become anchorable
// children of the referring array.
type Inner struct {
	Code string `json:"code"`
}

// Shapes carries the three nested shapes whose element / value properties must
// anchor under the correct items / additionalProperties pointers.
//
// swagger:model Shapes
type Shapes struct {
	// List is an inline array-of-struct: the anonymous element's properties
	// anchor under /definitions/Shapes/properties/list/items.
	List []struct {
		Tag string `json:"tag"`
	} `json:"list"`

	// Dict is an inline map-of-struct: the anonymous value's properties anchor
	// under /definitions/Shapes/properties/dict/additionalProperties.
	Dict map[string]struct {
		Val string `json:"val"`
	} `json:"dict"`

	// Over collapses the Go []string to an inline array-of-Inner via the
	// override; Inner's properties anchor under
	// /definitions/Shapes/properties/over/items.
	//
	// swagger:type []Inner
	Over []string `json:"over"`
}

// PatternPaths carries a patternProperties whose regex itself contains the
// RFC 6901 special characters '/' and '~'. The cross-ref pointer to a
// patternProperties value node is .../patternProperties/<regex>, with the regex
// segment escaped (~ → ~0, / → ~1) so it round-trips against the raw regex used
// as the JSON map key. The pre-fix code pointed these at additionalProperties.
//
// swagger:model PatternPaths
// swagger:patternProperties "^/api/~v[0-9]+": string
type PatternPaths struct {
	ID string `json:"id"`
}
