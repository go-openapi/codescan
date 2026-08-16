// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_default exercises a route with only a
// `default:` response.
package routes_responses_default

// GenericError is the catch-all error response.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		Message string `json:"message"`
	}
}

// Health swagger:route GET /health ops health
//
// Health probe.
//
// Responses:
//
//	default: genericError
func Health() {}
