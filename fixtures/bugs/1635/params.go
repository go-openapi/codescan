// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1635

// PositionParams anonymously embeds a named body struct marked in:body. This is
// the parameters counterpart of #1635: the embed IS the single body parameter
// (schema $ref'ing the embedded struct), not one body param per promoted field
// (which would be invalid — an operation allows at most one body parameter).
//
// swagger:parameters setPosition
type PositionParams struct {
	// in: body
	PositionResponseBody
}

// NamedBodyParams is the recommended form (a named Body field) — a green guard
// rail showing the correct single body parameter is produced.
//
// swagger:parameters namedSetPosition
type NamedBodyParams struct {
	// in: body
	Body PositionResponseBody
}

// SetPosition uses the anonymous-embed body params.
//
// swagger:route POST /position things setPosition
//
// Set position.
//
//	Responses:
//	  200: description: ok
func SetPosition() {}

// NamedSetPosition uses the named-Body-field params (guard rail).
//
// swagger:route POST /position-named things namedSetPosition
//
// Set position (named).
//
//	Responses:
//	  200: description: ok
func NamedSetPosition() {}
