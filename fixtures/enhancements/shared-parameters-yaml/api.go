// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters_yaml (Fixture 4) witnesses that the
// swagger:operation wholesale-YAML path runs the same shared-namespace
// checks: a $ref into #/parameters or #/responses is validated, and a
// dangling ref raises a diagnostic.
//
// See .claude/plans/features/shared-parameters-fixtures.md.
package shared_parameters_yaml

// CommonHeaders registers #/parameters/X-Request-ID.
//
// swagger:parameters *
type CommonHeaders struct {
	// in: header
	RequestID string `json:"X-Request-ID"`
}

// ErrorResponse registers #/responses/ErrorResponse.
//
// swagger:response *
type ErrorResponse struct {
	// in: body
	Body struct {
		// Message is a human-readable error message.
		Message string `json:"message"`
	} `json:"body"`
}

// OpA references the shared namespace from wholesale YAML; both refs
// resolve and are kept.
//
// swagger:operation GET /a opA
//
// ---
// summary: resolving refs
// parameters:
//   - $ref: '#/parameters/X-Request-ID'
// responses:
//   default:
//     $ref: '#/responses/ErrorResponse'
func OpA() {}

// OpB references names that do not exist in the shared namespace; each
// raises a dangling-ref diagnostic.
//
// swagger:operation GET /b opB
//
// ---
// summary: dangling refs
// parameters:
//   - $ref: '#/parameters/DoesNotExist'
// responses:
//   default:
//     $ref: '#/responses/Missing'
func OpB() {}
