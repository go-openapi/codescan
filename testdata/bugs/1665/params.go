// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1665

// idParams carries TWO swagger:parameters lines, binding the shared id path
// parameter to several operations across multiple annotation lines.
//
// swagger:parameters getFooByID getFooFooByID
// swagger:parameters getBarByID getBarBarByID
type idParams struct {
	// in: path
	ID string `json:"id"`
}

// GetFoo uses getFooByID.
//
// swagger:route GET /foo/{id} foo getFooByID
//
// Get a foo.
//
//	Responses:
//	  200: description: ok
func GetFoo() {}

// GetBar uses getBarByID.
//
// swagger:route GET /bar/{id} bar getBarByID
//
// Get a bar.
//
//	Responses:
//	  200: description: ok
func GetBar() {}
