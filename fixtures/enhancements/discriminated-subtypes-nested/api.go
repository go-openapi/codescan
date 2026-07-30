// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package discriminatednested exercises a MULTI-LEVEL polymorphic hierarchy: a
// subtype of a discriminated base that is itself a discriminated base
// (go-swagger#1913).
//
// A single route references Shape. The closure has to cascade:
//
//	Shape (discriminated)          <- the only referenced type
//	 |- Circle                     (leaf struct)
//	 `- Polygon (discriminated)    <- a subtype AND a base
//	     |- Square                 (leaf struct)
//	     `- Triangle               (leaf struct, pulls Coords)
//
// Reaching Polygon requires the reverse index; reaching Square / Triangle
// requires it to run AGAIN on a type that was itself only just pulled in. Coords
// then proves ordinary $ref discovery still works from a second-level leaf.
package discriminatednested

// Shape is the root of the hierarchy.
//
// swagger:model Shape
type Shape interface {
	// The kind of shape
	//
	// discriminator: true
	// swagger:name shapeType
	ShapeType() string

	// Area of the shape
	// swagger:name area
	Area() float64
}

// Polygon is a subtype of Shape AND a discriminated base of its own.
//
// swagger:model Polygon
type Polygon interface {
	// swagger:allOf
	Shape

	// The kind of polygon
	//
	// discriminator: true
	// swagger:name polygonType
	PolygonType() string

	// Number of sides
	// swagger:name sides
	Sides() int64
}

// Circle is a first-level leaf: a subtype of Shape directly.
//
// swagger:model Circle
type Circle struct {
	// swagger:allOf
	Shape

	// Radius of the circle
	Radius float64 `json:"radius"`
}

// Square is a second-level leaf, reachable only once Polygon has itself been
// pulled in.
//
// swagger:model Square
type Square struct {
	// swagger:allOf
	Polygon

	// Length of a side
	Side float64 `json:"side"`
}

// Triangle is a second-level leaf that pulls a further model of its own.
//
// swagger:model Triangle
type Triangle struct {
	// swagger:allOf
	Polygon

	// Corner coordinates
	Corners []Coords `json:"corners"`
}

// Coords is reached from Triangle only — through an ordinary $ref, two levels
// down the hierarchy.
//
// swagger:model Coords
type Coords struct {
	// X coordinate
	X float64 `json:"x"`

	// Y coordinate
	Y float64 `json:"y"`
}

// Unrelated is a swagger:model nothing references, in a hierarchy-bearing
// package: the negative control.
//
// swagger:model Unrelated
type Unrelated struct {
	V string `json:"v"`
}

// shapeResp carries the root of the hierarchy in its body.
//
// swagger:response shapeResp
type shapeResp struct {
	// in: body
	Body Shape `json:"body"`
}

// handler is the only route.
//
// swagger:route GET /shapes listShapes
//
// Lists shapes.
//
// responses:
//
//	200: shapeResp
func handler() {}
