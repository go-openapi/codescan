// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_path exercises a single path parameter declared
// via the swagger:route `+ name:` form.
package routes_params_path

// GetItem swagger:route GET /items/{id} items getItem
//
// Get an item by id.
//
// Parameters:
//   + name: id
//     in: path
//     description: item identifier
//     required: true
//     type: integer
//
// Responses:
//
//	200: description: OK
func GetItem() {}
