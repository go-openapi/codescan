// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_body_array_nested exercises nested-array body
// types: `type: [][]Item`.
package routes_params_body_array_nested

// Item is one item.
//
// swagger:model
type Item struct {
	Name string `json:"name"`
}

// SubmitMatrix swagger:route POST /matrix items submitMatrix
//
// Submit a matrix of items.
//
// Parameters:
//   + name: body
//     in: body
//     description: nested grid of items
//     required: true
//     type: [][]Item
//
// Responses:
//
//	200: description: OK
func SubmitMatrix() {}
