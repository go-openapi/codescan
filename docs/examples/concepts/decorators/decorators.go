// SPDX-License-Identifier: Apache-2.0

// Package decorators holds the annotated declarations used by the "Other type
// decorators" tutorial. decorators_test.go scans this package and writes the
// per-feature golden fragments the tutorial renders.
package decorators

// snippet:readonly

// Token is issued by the server.
//
// swagger:model
type Token struct {
	// ID is assigned by the server and cannot be set by clients.
	//
	// read only: true
	ID string `json:"id"`

	// Value is the token value.
	Value string `json:"value"`
}

// endsnippet:readonly

// snippet:deprecated

// swagger:route GET /legacy/ping legacy ping
//
// Ping is the legacy health check.
//
// deprecated: true
//
// responses:
//
//	200: pingResponse

// endsnippet:deprecated

// PingResponse is the body returned by ping.
//
// swagger:response pingResponse
type PingResponse struct {
	// in: body
	Body struct {
		// OK is true when healthy.
		OK bool `json:"ok"`
	}
}

// snippet:deprecatedmodel

// Gadget is a deprecated model. OpenAPI 2.0 has no native `deprecated` on a
// schema, so codescan emits `x-deprecated: true` — here triggered by the
// godoc-style "Deprecated:" paragraph, which is recognised on its own without a
// separate annotation. (The explicit `deprecated: true` annotation, shown on the
// operation above, has the same effect on a model or field.)
//
// Deprecated: superseded by the v2 widget API.
//
// swagger:model
type Gadget struct {
	// SerialNo is the legacy identifier.
	//
	// Deprecated: use the v2 identifier instead.
	SerialNo string `json:"serialNo"`

	// Name is the current display name.
	Name string `json:"name"`
}

// endsnippet:deprecatedmodel
