// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bodyfieldexample covers example: on a swagger:response struct Body
// field — array and scalar. Body-field schema keywords land on the body
// schema, not the discarded header (go-swagger#3013, #2942 family).
package bodyfieldexample

// ArrayBodyResp has an array Body field with an example.
//
// swagger:response ArrayBodyResp
type ArrayBodyResp struct {
	// in: body
	// example: ["a","b"]
	Body []string
}

// ScalarBodyResp has a scalar Body field with an example.
//
// swagger:response ScalarBodyResp
type ScalarBodyResp struct {
	// in: body
	// example: hello
	Body string
}

// swagger:route GET /array things getArrayBody
//
// responses:
//   200: ArrayBodyResp

// swagger:route GET /scalar things getScalarBody
//
// responses:
//   200: ScalarBodyResp
func handlers3() {}
