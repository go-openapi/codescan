// SPDX-License-Identifier: Apache-2.0

// Package nested holds the annotated declarations used by the "Multi-level
// hierarchies" section of the "Polymorphic models" tutorial. nested_test.go scans
// it and writes the golden fragments the tutorial renders, so the documentation
// can never drift from the scanner's real output.
package nested

// snippet:hierarchy

// Shape is the root of the hierarchy: a base written as an interface, whose
// `discriminator: true` member names the property a consumer switches on.
//
// swagger:model
type Shape interface {
	// ShapeType selects the concrete subtype.
	//
	// discriminator: true
	// required: true
	// swagger:name shapeType
	ShapeType() string

	// swagger:name area
	Area() float64
}

// Polygon is a subtype of Shape AND a base of its own: it composes Shape as an
// allOf member and declares a second discriminator. An intermediate level is
// written as an interface, because only an interface can be embedded by the
// concrete structs below it.
//
// swagger:model
type Polygon interface {
	// swagger:allOf
	Shape

	// PolygonType selects the concrete polygon.
	//
	// discriminator: true
	// required: true
	// swagger:name polygonType
	PolygonType() string
}

// Square is a leaf, two levels down. It composes the INTERMEDIATE type, so it
// inherits Shape transitively.
//
// swagger:model
type Square struct {
	// swagger:allOf
	Polygon

	// Side is the length of a side.
	Side float64 `json:"side"`
}

// endsnippet:hierarchy

// ShapeResponse returns the ROOT of the hierarchy — nothing in the API surface
// names Polygon or Square.
//
// swagger:response shapeResponse
type ShapeResponse struct {
	// in: body
	Body Shape `json:"body"`
}

// swagger:route GET /shapes shapes listShapes
//
// Lists shapes.
//
// responses:
//
//	200: shapeResponse
