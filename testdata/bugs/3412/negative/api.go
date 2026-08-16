// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package negative reproduces go-swagger issue #3412 ("negative enum values
// are dropped"): a signed constant such as `PanLeft PanDirection = -1` is not
// a literal in the Go grammar — it reaches the AST as a unary expression
// wrapping the literal — so the enum member was silently skipped and only the
// non-negative values survived.
//
// The enum type is consumed from every enum-carrying target — a response
// header, a schema property (in: body) and a non-body parameter (in: query) —
// so the value must survive the per-target coercion, not just the scan.
//
// A float enum covers the FLOAT literal branch, and a non-decimal enum covers
// the base-detected integer forms (hex, binary, octal, digit separators).
package negative

// swagger:route GET /pan/{pan} pan setPan
//
// Sets the pan direction.
//
// responses:
//
//	200: ControlParams
func SetPan() {}

// PanDirection is a signed enum: its members straddle zero.
//
// swagger:enum PanDirection
type PanDirection int8

const (
	// PanLeft pans to the left.
	PanLeft PanDirection = -1

	// NoPan holds the current position.
	NoPan PanDirection = 0

	// PanRight pans to the right.
	PanRight PanDirection = +1
)

// Tilt is a signed float enum.
//
// swagger:enum Tilt
type Tilt float64

const (
	TiltDown Tilt = -0.5
	TiltFlat Tilt = 0
	TiltUp   Tilt = 0.5
)

// Mask is an integer enum written in the non-decimal Go literal forms.
//
// swagger:enum Mask
type Mask int64

const (
	MaskHex        Mask = 0x2a
	MaskBinary     Mask = 0b101010
	MaskOctal      Mask = 0o52
	MaskUnderscore Mask = 1_000
)

// Zoom is a float32 enum whose FIRST member is written as an integer literal.
//
// Typing the schema from the first parsed value made this `{type: integer}` with fractional
// members — a schema no validator can satisfy — and moving ZoomNone down the block silently
// changed the emitted type. Both are settled by typing from the declared Go type instead.
//
// swagger:enum Zoom
type Zoom float32

const (
	ZoomNone Zoom = 0
	ZoomOut  Zoom = -1.5
	ZoomIn   Zoom = 1.5
)

// Aperture is an unsigned enum spanning the full uint64 range: its top member does not fit an
// int64, the signed parse the scanner tries first.
//
// swagger:enum Aperture
type Aperture uint64

const (
	ApertureClosed Aperture = 0
	ApertureOpen   Aperture = 18446744073709551615
)

// ControlParams carries the enums as response headers.
//
// PTZ control parameters
//
// swagger:response
type ControlParams struct {
	// specifies the direction of the pan. -1 is pan left, 0 is no pan, 1 is pan right.
	//
	// in: header
	Pan PanDirection `json:"pan,omitempty"`

	// specifies the tilt angle.
	//
	// in: header
	Tilt Tilt `json:"tilt,omitempty"`
}

// ControlState carries the enums as schema properties.
//
// swagger:model ControlState
type ControlState struct {
	// The pan direction.
	Pan PanDirection `json:"pan"`

	// The tilt angle.
	Tilt Tilt `json:"tilt"`

	// The active mask.
	Mask Mask `json:"mask"`

	// The zoom step.
	Zoom Zoom `json:"zoom"`

	// The aperture.
	Aperture Aperture `json:"aperture"`
}

// PanParams carries the enum as a query parameter.
//
// swagger:parameters setPan
type PanParams struct {
	// The requested pan direction.
	//
	// in: query
	Pan PanDirection `json:"pan"`
}
