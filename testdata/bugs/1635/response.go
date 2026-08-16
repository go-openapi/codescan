// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1635

// PositionResponseBody is the named body struct.
type PositionResponseBody struct {
	// X Position
	Positionx string `json:"positionx"`
	// Y Position
	Positiony string `json:"positiony"`
}

// PositionResponse embeds a NAMED body struct anonymously with in:body. This is
// the #1635 case: the body should be the embedded struct's object schema (or a
// $ref), but the scanner emits schema {type: string}.
//
// swagger:response positionResponse
type PositionResponse struct {
	// in: body
	PositionResponseBody
}

// NamedBodyResponse is the recommended form (a named Body field) — included as a
// green guard rail showing the correct object schema is produced.
//
// swagger:response namedBodyResponse
type NamedBodyResponse struct {
	// in: body
	Body PositionResponseBody
}
