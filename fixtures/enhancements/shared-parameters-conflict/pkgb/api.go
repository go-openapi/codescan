// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package pkgb is the SECOND package of Fixture 3 (cross-package
// conflicts). Its shared parameter X-Token and shared response
// ErrorResponse collide on short name with pkga's; both are dropped with
// a keep-first warning (no rename — short-name refs must stay valid).
//
// See .claude/plans/features/shared-parameters-fixtures.md.
package pkgb

// TokenB collides with pkga's #/parameters/X-Token on short name, with a
// DIFFERENT `in:` (query vs header). Expected: dropped + warning.
//
// swagger:parameters *
type TokenB struct {
	// in: query
	Token string `json:"X-Token"`
}

// ErrorResponse collides with pkga's #/responses/ErrorResponse on short
// name, with a different body shape. Expected: dropped + warning.
//
// swagger:response *
type ErrorResponse struct {
	// in: body
	Body struct {
		// Message is a human-readable error message.
		Message string `json:"message"`
	} `json:"body"`
}

// ListB gives the scan an operation in pkgb.
//
// swagger:route GET /b beta listB
// Responses:
//
//	default: ErrorResponse
func ListB() {}
