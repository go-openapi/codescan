// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package description_title_override_responses witnesses the swagger:description
// override on responses and response headers (P3 of the override feature, Q30
// close-out), plus the swagger:title context error: OpenAPI 2.0 Response and
// Header objects have no `title` field, so a swagger:title on a response /
// header is rejected with scan.context-invalid and the description override is
// still applied.
//
// See .claude/plans/features/swagger-description-override-design.md.
package description_title_override_responses

// ErrorBody is the error payload returned in the response body.
//
// swagger:model
type ErrorBody struct {
	Message string `json:"message"`
}

// ErrorResponse is the Go-facing response doc, written for Go readers — it
// should not leak into the API spec.
//
// swagger:response errorResponse
// swagger:description The error payload returned to API consumers.
// swagger:title Ignored — responses have no title
type ErrorResponse struct {
	// XErrorCode is the Go-facing header doc.
	//
	// swagger:description The machine-readable error code.
	XErrorCode string `json:"X-Error-Code"`

	// ErrorBody carries the structured error.
	//
	// in: body
	Body ErrorBody `json:"body"`
}
