// SPDX-License-Identifier: Apache-2.0

// Package enums backs the "Enumerations" tutorial: one Go const block per
// enum-shaping rule, each paired with the fragment the scanner emits from it.
package enums

import "github.com/go-openapi/strfmt"

// snippet:iota

// Weekday is an iota enum: only the first spec carries a type and a value, and
// every following one inherits both implicitly.
//
// swagger:enum Weekday
type Weekday int

const (
	// Sunday is the first day.
	Sunday Weekday = iota
	// Monday is the second day.
	Monday
	// Tuesday is the third day.
	Tuesday
)

// Schedule carries the enum, which is what makes it reachable and so emitted.
//
// swagger:model
type Schedule struct {
	// Day the job runs on.
	Day Weekday `json:"day"`
}

// endsnippet:iota

// snippet:expressions

// Level is built from a constant expression and from a reference to an earlier
// member — neither of which is a literal.
//
// swagger:enum Level
type Level int

const (
	// LevelLow is the floor.
	LevelLow Level = 1 << 3
	// LevelHigh doubles it.
	LevelHigh Level = LevelLow * 2
)

// Threshold carries the computed enum.
//
// swagger:model
type Threshold struct {
	// Level to alert at.
	Level Level `json:"level"`
}

// endsnippet:expressions

// snippet:signed

// PanDirection straddles zero. A signed constant is not a literal in the Go
// grammar — it is an expression wrapping one — so these are the members that
// used to go missing.
//
// swagger:enum PanDirection
type PanDirection int8

const (
	// PanLeft pans to the left.
	PanLeft PanDirection = -1
	// NoPan holds the current position.
	NoPan PanDirection = 0
	// PanRight pans to the right.
	PanRight PanDirection = 1
)

// Camera carries the signed enum, declared int8.
//
// swagger:model
type Camera struct {
	// Pan direction of the camera.
	Pan PanDirection `json:"pan"`
}

// endsnippet:signed

// snippet:width

// Zoom is a float32 enum whose FIRST member is written as an integer literal.
// The schema type follows the declared type, so the block can be reordered
// freely.
//
// swagger:enum Zoom
type Zoom float32

const (
	// ZoomNone is the neutral step.
	ZoomNone Zoom = 0
	// ZoomOut steps back.
	ZoomOut Zoom = -1.5
	// ZoomIn steps in.
	ZoomIn Zoom = 1.5
)

// Lens carries the float enum.
//
// swagger:model
type Lens struct {
	// Zoom step of the lens.
	Zoom Zoom `json:"zoom"`
}

// endsnippet:width

// snippet:strfmt

// Kind is an enum written over a string format rather than over a plain string:
// the format of the type it is declared over comes with it.
//
// swagger:enum Kind
type Kind strfmt.UUID

const (
	// KindPrimary is the primary kind.
	KindPrimary Kind = "0a8bcf1e-0000-0000-0000-000000000000"
	// KindSecondary is the secondary kind.
	KindSecondary Kind = "0a8bcf1e-1111-1111-1111-111111111111"
)

// Label carries the formatted enum.
//
// swagger:model
type Label struct {
	// Kind of the label.
	Kind Kind `json:"kind"`
}

// endsnippet:strfmt

// snippet:runes

// Letter is a rune enum. A rune is an int32, on the wire as much as in Go, so
// the members are code points — 'a' is 97.
//
// swagger:enum Letter
type Letter rune

const (
	// LetterA is the first letter.
	LetterA Letter = 'a'
	// LetterB is the second letter.
	LetterB Letter = 'b'
)

// Glyph carries the rune enum.
//
// swagger:model
type Glyph struct {
	// Letter of the glyph.
	Letter Letter `json:"letter"`
}

// endsnippet:runes

// swagger:route GET /cameras/search cameras search
//
// Searches cameras by pan direction.
//
// responses:
//
//	200: cameraList

// CameraList is the response body.
//
// swagger:response cameraList
type CameraList struct {
	// in: body
	Body []Camera `json:"body"`
}

// snippet:params

// SearchParams consumes an enum from a non-body parameter, where OpenAPI 2.0
// forbids a $ref: the members and the format ship inline.
//
// swagger:parameters search
type SearchParams struct {
	// Direction to pan while searching.
	//
	// in: query
	Pan PanDirection `json:"pan"`

	// Directions the client accepts.
	//
	// in: query
	Accepted []PanDirection `json:"accepted"`
}

// endsnippet:params
