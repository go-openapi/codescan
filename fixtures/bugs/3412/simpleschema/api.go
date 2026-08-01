// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package simpleschema pins the propagation surface of `swagger:enum`: every place an enum type
// can be consumed must carry the members AND the type/format of the declared Go type.
//
// The SimpleSchema targets are the interesting half. OAS v2 forbids `$ref` on a non-body parameter
// or a response header, so the enum ships inline there through a different builder path than a
// schema property — one per `in:` location, plus the `items` of an array-typed one. A regression in
// any single target is invisible to a definitions-only assertion, which is why this fixture is
// pinned by a whole-spec golden rather than by property lookups.
package simpleschema

// swagger:route GET /pan/{pan} pan setPan
//
// Sets the pan direction.
//
// responses:
//
//	200: PanResponse
func SetPan() {}

// swagger:route POST /pan upload uploadPan
//
// Uploads a pan setting.
//
// responses:
//
//	200: PanResponse
func UploadPan() {}

// PanDirection is a signed int8 enum: negative member (#3412) and a width the members themselves
// cannot carry (they all arrive as int64).
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

// PanParams consumes the enum from every non-body parameter location.
//
// swagger:parameters setPan
type PanParams struct {
	// in: path
	Pan PanDirection `json:"pan"`

	// in: query
	Preferred PanDirection `json:"preferred"`

	// in: header
	Fallback PanDirection `json:"fallback"`

	// in: query
	Allowed []PanDirection `json:"allowed"`
}

// UploadParams consumes the enum from a form parameter.
//
// swagger:parameters uploadPan
type UploadParams struct {
	// in: formData
	Requested PanDirection `json:"requested"`
}

// PanResponse consumes the enum from response headers — scalar and array — and from a body schema.
//
// swagger:response
type PanResponse struct {
	// in: header
	Applied PanDirection `json:"applied"`

	// in: header
	Rejected []PanDirection `json:"rejected"`

	// in: body
	Body PanState `json:"body"`
}

// PanState consumes the enum from a schema property, scalar and array.
//
// swagger:model PanState
type PanState struct {
	// The current direction.
	Current PanDirection `json:"current"`

	// The directions still available.
	Available []PanDirection `json:"available"`
}
