// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_full_petstore_shape exercises a swagger:route block
// using the full petstore-style metadata surface (Consumes / Produces /
// Schemes / Security / Tags / Parameters / Responses / Extensions)
// alongside the `+ name:` parameter sub-grammar.
package routes_full_petstore_shape

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GenericError is the catch-all error response.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		Message string `json:"message"`
	}
}

// CreatePet swagger:route POST /pets pets users createPetFull
//
// Create a pet with the full metadata surface.
//
// Consumes:
//   - application/json
//   - application/x-protobuf
//
// Produces:
//   - application/json
//
// Schemes: http, https
//
// Security:
//
//	api_key:
//	oauth: read, write
//
// Parameters:
//   + name: trace
//     in: query
//     description: enable tracing
//     type: boolean
//     default: false
//   + name: body
//     in: body
//     description: pet to create
//     required: true
//     type: Pet
//
// Responses:
//
//	201: body:Pet the created pet
//	default: response:genericError
//
// Extensions:
//
//	x-route-flag: true
func CreatePet() {}
