// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_array exercises array-shaped body responses,
// including the nested `[][]` form.
package routes_responses_array

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Item is one item.
//
// swagger:model
type Item struct {
	Name string `json:"name"`
}

// ListPets swagger:route GET /pets pets listPetsArray
//
// List pets and a matrix of items.
//
// Responses:
//
//	200: body:[]Pet the pet list
//	202: body:[][]Item items grid
func ListPets() {}
