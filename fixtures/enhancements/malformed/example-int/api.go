// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package example_int carries a parameter with an `example:` value that
// does not parse as the declared int type, exercising setExample.Parse's
// error branch.
package example_int

// BadParams has an int query parameter with a non-numeric example.
//
// swagger:parameters badExampleInt
type BadParams struct {
	// in: query
	// example: notanumber
	Value int `json:"value"`
}

// DoBad swagger:route GET /bad-example bad badExampleInt
//
// Responses:
//
//	200: description: OK
func DoBad() {}
