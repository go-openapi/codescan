// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_body_array exercises a body parameter that is
// an array of a referenced model: `type: []Pet`.
package routes_params_body_array

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreatePets swagger:route POST /pets pets createPets
//
// Create several pets.
//
// Parameters:
//   + name: body
//     in: body
//     description: pets to create
//     required: true
//     type: []Pet
//
// Responses:
//
//	201: description: created
func CreatePets() {}
