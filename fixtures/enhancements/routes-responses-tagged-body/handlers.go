// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_tagged_body exercises response lines carrying
// the `body:` tag with a positional description tail.
package routes_responses_tagged_body

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetPet swagger:route GET /pets/{id} pets getPet
//
// Get a pet by id.
//
// Responses:
//
//	200: body:Pet the pet as returned
//	404: description: not found
func GetPet() {}
