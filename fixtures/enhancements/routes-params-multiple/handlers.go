// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_multiple exercises three chained parameters
// mixing in: values (path, query, body).
package routes_params_multiple

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// UpdatePet swagger:route PUT /pets/{id} pets updatePet
//
// Update a pet identified by id.
//
// Parameters:
//   + name: id
//     in: path
//     description: pet identifier
//     required: true
//     type: integer
//   + name: dryRun
//     in: query
//     description: validate without persisting
//     type: boolean
//     default: false
//   + name: body
//     in: body
//     description: updated pet
//     required: true
//     type: Pet
//
// Responses:
//
//	200: description: OK
func UpdatePet() {}
