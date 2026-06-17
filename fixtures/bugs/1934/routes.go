// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1934

// Handler holds a route whose response/params types are declared INSIDE a
// function body (the #1934 layout).
func Handler() {
	// swagger:route GET /pets pets getPets
	//
	// Responses:
	//   default: body:someResponse
}

func getPets() {
	// swagger:parameters getPets
	type params struct {
		Foo string `json:"foo"`
	}
	_ = params{}

	// swagger:model someResponse
	type response struct {
		Name string `json:"name"`
	}
	_ = response{}
}
