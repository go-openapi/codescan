// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package response_name_keyword exercises the name keyword on a response
// struct field. It renames the response header (the Headers map key),
// overriding the json-tag / Go-field derivation, mirroring the same
// keyword on a parameters field. Being a structural keyword it is also
// stripped from the header description rather than leaking into it as
// prose.
package response_name_keyword

// swagger:model Widget
type Widget struct {
	Color string `json:"color"`
}

// swagger:response widgetResponse
type widgetResponse struct {
	// requests remaining in the window
	// name: X-Rate-Limit
	// minimum: 0
	RateLimit int64

	// the response payload
	// in: body
	Body Widget
}
