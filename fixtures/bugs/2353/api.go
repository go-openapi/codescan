// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2353

// Req is the body model.
//
// swagger:model
type Req struct {
	Name string `json:"name"`
}

// swagger:parameters doSomething
type doSomethingParams struct {
	// in: path
	// required: true
	ID string `json:"id"`
	// in: body
	Request Req
}

// swagger:route POST /test/{id} test doSomething
//
// Do something.
//
// responses:
//
//	200: description: ok
func DoSomething() {}
