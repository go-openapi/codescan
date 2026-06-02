// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_body_ref exercises a body parameter referencing
// a swagger:model.
package routes_params_body_ref

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreatePet swagger:route POST /pets pets createPet
//
// Create a new pet.
//
// Parameters:
//   + name: body
//     in: body
//     description: pet to create
//     required: true
//     type: Pet
//
// Responses:
//
//	201: description: created
func CreatePet() {}
