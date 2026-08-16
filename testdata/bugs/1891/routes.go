// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1891

type (
	// FooOutput for output
	//
	// swagger:model FooOutput
	FooOutput struct {
		Addr string `json:"addr"`
	}
)

// FooInput for input
//
// swagger:parameters fooInput
type FooInput struct {
	// in: query
	App string `json:"app"`
	// in: query
	UID string `json:"uid"`
}

// Foo declares a route inside a function body, alongside a grouped type
// declaration (the #1891 "group type" layout).
func Foo() {
	// swagger:route GET /foo foo fooInput
	//
	// get socket address
	//
	// Responses:
	//   200: body:FooOutput
}
