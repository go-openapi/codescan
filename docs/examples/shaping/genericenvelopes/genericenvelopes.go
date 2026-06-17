// SPDX-License-Identifier: Apache-2.0

// Package genericenvelopes holds the annotated declarations for the
// "Documenting generic responses" how-to. genericenvelopes_test.go scans it and
// writes the golden definitions and path items the guide renders, so the
// documentation can never drift from the scanner's real output.
//
// The scenario is go-swagger#2596: a single generic envelope is returned by
// every handler, but the spec should describe a concrete payload per operation.
package genericenvelopes

// snippet:envelope

// APIResponse is the one generic envelope every handler returns. Because Data is
// an open type (any, i.e. interface{}), the scanner can only render it as an
// open schema — it has no way to know which concrete payload a given operation
// puts there.
//
// swagger:model
type APIResponse struct {
	Status  string `json:"status"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

// endsnippet:envelope

// StatusReport is the concrete payload returned by GET /status.
//
// swagger:model
type StatusReport struct {
	Code    int    `json:"code"`
	Detail  string `json:"detail"`
	Healthy bool   `json:"healthy"`
}

// UserSummary is the concrete payload returned by GET /users/{id}.
//
// swagger:model
type UserSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// snippet:docstruct

// StatusEnvelope documents the /status response: it embeds APIResponse and
// shadows the open Data with the concrete StatusReport. Handlers keep returning
// APIResponse — this type just gives the scanner a concrete shape.
//
// swagger:model
type StatusEnvelope struct {
	APIResponse

	// Data carries the concrete status report.
	Data StatusReport `json:"data"`
}

// endsnippet:docstruct

// UserEnvelope specialises the same envelope for the /users response with a
// different payload — one envelope shape, a concrete spec per operation.
//
// swagger:model
type UserEnvelope struct {
	APIResponse

	// Data carries the concrete user summary.
	Data UserSummary `json:"data"`
}

// snippet:routes

// swagger:route GET /status status getStatus
//
// Returns the service status wrapped in the generic envelope. The response is
// declared as the doc-only StatusEnvelope, so data is the concrete StatusReport
// rather than an open schema.
//
//	Responses:
//	  200: body:StatusEnvelope the status envelope

// swagger:route GET /users/{id} users getUser
//
// Returns a user wrapped in the same envelope, specialised to UserSummary.
//
//	Responses:
//	  200: body:UserEnvelope the user envelope

// endsnippet:routes
