// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package pkga is the FIRST package of Fixture 3 (cross-package
// conflicts). By import-path order it is the survivor of the duplicate
// shared short names also declared in pkgb.
//
// See .claude/plans/features/shared-parameters-fixtures.md.
package pkga

// TokenA registers #/parameters/X-Token (header). pkgb declares the same
// short name with a different `in:` — pkga wins, pkgb is dropped + warned.
//
// swagger:parameters *
type TokenA struct {
	// in: header
	Token string `json:"X-Token"`
}

// StatusParam registers #/parameters/Status. The model Status below
// registers #/definitions/Status — independent namespaces, NO conflict.
//
// swagger:parameters *
type StatusParam struct {
	// in: header
	Status string `json:"Status"`
}

// Status is a model at #/definitions/Status, coexisting with the shared
// parameter #/parameters/Status (cross-namespace non-conflict witness).
//
// swagger:model
type Status struct {
	// State of the resource.
	State string `json:"state"`
}

// ErrorResponse force-registers #/responses/ErrorResponse. pkgb declares
// the same response short name with a different body — pkga wins.
//
// swagger:response *
type ErrorResponse struct {
	// in: body
	Body struct {
		// Code is a machine-readable error code.
		Code int `json:"code"`
	} `json:"body"`
}

// ListA gives the scan an operation in pkga.
//
// swagger:route GET /a alpha listA
// Responses:
//
//	default: ErrorResponse
func ListA() {}
