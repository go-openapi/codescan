// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package header_extensions exercises M2's new Walker.Extension wiring
// on the response-header path. Pre-M2, the responses bridge had no
// extension handling on headers at all — a user-authored
// `Extensions:` block on a header field was silently dropped.
// Post-M2, the bridge runs the same Walker.Extension callback the
// parameter bridge uses, so `x-*` entries land on the
// header.Extensions map.
package header_extensions

// ExtendedResponse demonstrates a response header carrying a
// user-authored vendor extension via the `Extensions:` block. M2's
// bridge dispatches the typed extension value onto the header.
//
// swagger:response extendedResponse
type ExtendedResponse struct {
	// RateLimit advertises the remaining quota.
	//
	// in: header
	// Extensions:
	//   x-rate-window: 60s
	//   x-burst-allowed: true
	RateLimit int `json:"X-Rate-Limit"`

	// Body is the canonical response payload.
	//
	// in: body
	Body struct {
		// Message carries the response message.
		Message string `json:"message"`
	} `json:"body"`
}

// DoExtended handles the route.
//
// swagger:route GET /extended ext extendedResponse
//
// Responses:
//
//	200: extendedResponse
func DoExtended() {}
