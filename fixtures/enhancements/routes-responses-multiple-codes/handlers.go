// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_multiple_codes exercises a route with
// default + 2xx + 4xx + 5xx response codes in one block.
package routes_responses_multiple_codes

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

// GetPet swagger:route GET /pets/{id} pets getPetMultiCode
//
// Get a pet.
//
// Responses:
//
//	200: body:Pet the pet
//	404: description: not found
//	500: description: server error
//	default: response:genericError
func GetPet() {}
