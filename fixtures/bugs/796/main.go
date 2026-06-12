// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// A Bug API
//
// An API Definition to show a bug in go-swagger.
//
// swagger:meta
package bug796

// Ping Response
//
// swagger:model pingResponse
type pingResponse struct {
}

// Handler carries no swagger:model annotation and is referenced by
// nothing in the API: under `-m` it must NOT become a definition.
type Handler interface {
	Foo() int
}

// swagger:parameters ping
type pingParams struct {
	// Represents who is pinging
	//
	// in: path
	// required: true
	Who string `json:"who"`
}

// swagger:route GET /ping/{who} ping
//
// Test your connection with this service.
//
//	Produces:
//	  plain/text
//
//	Responses:
//	  200: body:pingResponse
func ping() {
}
