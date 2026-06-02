// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_query_validations exercises query parameters
// carrying SimpleSchema validations (min, max, default, enum).
package routes_params_query_validations

// ListItems swagger:route GET /items items listItems
//
// List items filtered by parameters.
//
// Parameters:
//   + name: limit
//     in: query
//     description: maximum number of items returned
//     type: integer
//     min: 1
//     max: 100
//     default: 20
//   + name: status
//     in: query
//     description: filter by status
//     type: string
//     enum: pending,active,archived
//
// Responses:
//
//	200: description: OK
func ListItems() {}
