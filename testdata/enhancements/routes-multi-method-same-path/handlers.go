// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_multi_method_same_path exercises two handlers on the
// same path (GET + POST) with different parameter/response shapes —
// confirms the routes builder merges multiple operations under one
// path entry without cross-contamination.
package routes_multi_method_same_path

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ListPets swagger:route GET /pets pets listPetsMulti
//
// List pets.
//
// Parameters:
//   + name: limit
//     in: query
//     description: page size
//     type: integer
//     min: 1
//     max: 100
//
// Responses:
//
//	200: body:[]Pet pets
func ListPets() {}

// CreatePet swagger:route POST /pets pets createPetMulti
//
// Create a pet.
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
//	201: body:Pet the created pet
func CreatePet() {}
