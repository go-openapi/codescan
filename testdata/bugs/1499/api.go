// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1499

// Bar is a Go struct the user wants to expose as a plain string in a parameter,
// via a swagger:type override (go-swagger#1499).
type Bar struct {
	X string
}

// swagger:parameters someOperation
type foo struct {
	// in: query
	// swagger:type string
	Bar Bar `json:"bar"`

	// A []-array override collapses the Go struct slice to a simple
	// array-of-string query parameter (go-swagger#1499).
	//
	// in: query
	// swagger:type []string
	Bars []Bar `json:"bars"`
}

// swagger:route GET /foo foo someOperation
//
// Foo.
//
// responses:
//
//	200: description: ok
func SomeOperation() {}
