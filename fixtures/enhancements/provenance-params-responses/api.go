// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package provparamsresp exercises cross-ref jsonpointer anchoring for the
// parameters and responses builders:
//
//   - a parameter anchors at /paths/{path}/{method}/parameters/{i} and stops
//     there (no drill-down into a body parameter's schema — the array index is
//     only known after path binding);
//   - a response header anchors at /responses/{name}/headers/{h};
//   - an in:body response field's inline struct anchors its properties at
//     /responses/{name}/schema/properties/{f}.
package provparamsresp

// provParams is the query/path parameter set for provOp.
//
// swagger:parameters provOp
type provParams struct {
	// in: query
	Limit int64 `json:"limit"`
}

// provResp is a response carrying both a header and an inline-struct body, so
// both anchor kinds fire.
//
// swagger:response provResp
type provResp struct {
	// in: header
	XRequestID string `json:"X-Request-Id"`

	// in: body
	Body struct {
		Status string `json:"status"`
	} `json:"body"`
}

// provHandler is the route. It references provResp so the response is bound to
// the operation, and provOp ties the parameter set to this path/method.
//
// swagger:route GET /prov provOp
//
// Prov.
//
// responses:
//
//	200: provResp
func provHandler() {}
