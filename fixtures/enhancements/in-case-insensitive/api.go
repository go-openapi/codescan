// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package in_case_insensitive witnesses Q29 — the `in:` value must
// be normalised case-insensitively against the closed OAS v2
// vocabulary (`query`, `path`, `header`, `body`, `formData`).
//
// Pre-fix (Q29): the scanner only recognised the exact lowercase
// canonical form. Mixed-case forms (`in: Body`, `in: QUERY`,
// `in: Header`, etc.) silently fell through every comparison,
// leaving the field at the default `query` location for parameters
// or producing an `invalidIn` diagnostic for responses. Go-swagger's
// own codegen emits capitalised forms (`in: Body`) so the
// regression broke compat with anyone scanning generated server
// stubs.
//
// Fix wires the three capture sites (parameters/doc_signals,
// responses/doc_signals, routebody/parameters) through a single
// `grammar.NormalizeIn` helper that lowercases + validates.
//
// This fixture pins both halves of the witness:
//
//  1. Parameters: every standard location declared with a non-
//     canonical case (`Body`, `QUERY`, `Path`, `Header`,
//     `FORMDATA`) — each must land at its canonical lowercase
//     location in the emitted spec.
//  2. Responses: a response with a mixed-case `Body` header — must
//     route through the body branch, not be flagged invalidIn.
package in_case_insensitive

import "io"

// HandleMixed is the operation that consumes the mixed-case
// parameters and emits the mixed-case response. Routes the witness
// into the spec's paths so the parameters can be inspected.
//
// swagger:route POST /mixed handlers mixedCaseParamsRequest
// Responses:
//
//	200: mixedCaseResponse
func HandleMixed() {}

// MixedCaseParams declares one of every parameter location with a
// non-canonical case spelling. The post-fix scanner must land each
// field at the canonical lowercase location.
//
// swagger:parameters mixedCaseParamsRequest
type MixedCaseParams struct {
	// in: Body
	// required: true
	Payload Payload `json:"payload"`

	// in: QUERY
	Search string `json:"search"`

	// in: Path
	// required: true
	ID string `json:"id"`

	// in: Header
	Token string `json:"X-Token"`

	// in: FORMDATA
	Upload io.Reader `json:"upload"`
}

// Payload is a small body struct exercised under in: Body.
type Payload struct {
	Note string `json:"note"`
}

// MixedCaseResponse declares a Body-shaped response with a
// mixed-case `in: Body` line — the response builder must route it
// through the body branch, not the header / invalidIn branch.
//
// swagger:response mixedCaseResponse
type MixedCaseResponse struct {
	// in: Body
	Body Payload `json:"body"`
}
