// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_mixed_bodies exercises a single route mixing
// body refs (schema) and response refs across status codes.
package routes_responses_mixed_bodies

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

// GetPet swagger:route GET /pets/{id} pets getPetMixed
//
// Get a pet.
//
// Responses:
//
//	200: body:Pet the pet
//	default: response:genericError
func GetPet() {}
