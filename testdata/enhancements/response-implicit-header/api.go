// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_implicit_header pins Q1's response-side fix: the
// `in:` annotation on a `swagger:response` Go struct's field is a
// scanner-side discriminator (OAS v2's wire format has no `in` on
// response headers) and an omitted line now explicitly defaults to
// "header" rather than falling through an incidental
// `in != "body"` check.
//
// The package exercises four variants captured by the integration
// test golden:
//
//   - empty struct → response with description only
//   - all-header struct (no `in: body`) → every field a header,
//     mixing implicit and explicit
//   - mixed struct → body + headers (implicit + explicit)
//   - invalid-`in:` struct → diagnostic emitted, field still becomes
//     a header (not silently ignored)
package response_implicit_header

// EmptyResponse — empty Go struct with a swagger:response
// annotation. Produces a response with description only; no Headers
// map, no Schema.
//
// swagger:response emptyResponse
type EmptyResponse struct{}

// AllHeadersResponse — every field becomes a header. Etag carries
// no `in:` line at all (default); RateLimit carries an explicit
// `in: header` (parity with existing fixtures).
//
// swagger:response allHeadersResponse
type AllHeadersResponse struct {
	// Etag carries an HTTP entity tag for cache validation.
	// No `in:` line — defaults to header (Q1 fix).
	Etag string `json:"ETag"`

	// RateLimit advertises the remaining quota.
	//
	// in: header
	RateLimit int `json:"X-Rate-Limit"`
}

// MixedResponse — body field plus headers (implicit + explicit).
//
// swagger:response mixedResponse
type MixedResponse struct {
	// Tag is a tracing tag. Implicit header (no `in:`).
	Tag string `json:"X-Tag"`

	// Limit advertises the remaining quota.
	//
	// in: header
	Limit int `json:"X-Limit"`

	// Body is the response payload.
	//
	// in: body
	Body struct {
		Message string `json:"message"`
	} `json:"body"`
}

// InvalidInResponse — non-vocabulary `in:` value. The scanner emits
// a CodeInvalidAnnotation warning naming the bad value, then falls
// back to the default (header).
//
// swagger:response invalidInResponse
type InvalidInResponse struct {
	// Cookie carries a session cookie. The `in: cookie` line is
	// not in the OAS v2 vocabulary; the scanner warns and treats
	// the field as a header anyway.
	//
	// in: cookie
	Cookie string `json:"Cookie"`
}
