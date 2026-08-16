// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package apsimpleschema is the safeguard witness: additionalProperties and
// patternProperties are object-schema keywords with no OAS2 SimpleSchema form,
// so on a non-body parameter or a response header they must be dropped with a
// CodeUnsupportedInSimpleSchema diagnostic — never silently absorbed.
package apsimpleschema

// SafeguardParams carries object-only keywords on a SimpleSchema query
// parameter.
//
// swagger:parameters doSafeguard
type SafeguardParams struct {
	// Filter is a query parameter.
	//
	// in: query
	// additionalProperties: true
	// patternProperties: ^x-
	Filter string `json:"filter"`
}

// SafeguardResp carries object-only keywords on a SimpleSchema response header.
//
// swagger:response safeguardResponse
type SafeguardResp struct {
	// XThing is a response header.
	//
	// in: header
	// additionalProperties: true
	// patternProperties: ^y-
	XThing string `json:"X-Thing"`
}

// swagger:route GET /safeguard doSafeguard
//
// responses:
//   200: safeguardResponse
func doSafeguard() {}
