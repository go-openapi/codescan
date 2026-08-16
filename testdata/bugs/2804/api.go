// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2804 reproduces go-swagger issue #2804 ("Should swagger:parameters
// support map[string][]string?"): a map-typed swagger:parameters field crashes
// the scan with a nil-pointer dereference in parameterBuilder.buildFromField
// (Schema.Typed on a nil schema), rather than producing a parameter or a
// diagnostic. A map is not representable as an OAS2 simple (query/formData)
// parameter.
//
// The same rule applies to the other OAS2 SimpleSchema target — response
// headers: a map-typed (non-body) response field has no SimpleSchema
// representation and must be skipped with a diagnostic rather than silently
// corrupting the response body schema.
package bug2804

// swagger:parameters doQuery
type Query struct {
	// in: query
	PropertyFilters map[string][]string `json:"property_filters"`
}

// Resp is a response carrying a map-typed header field, which is not
// representable as an OAS2 SimpleSchema header.
//
// swagger:response doQueryResponse
type Resp struct {
	// in: header
	Filters map[string][]string `json:"X-Filters"`
}

// swagger:route GET /q things doQuery
//
// responses:
//   200: doQueryResponse
func doQuery() {}
