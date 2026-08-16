// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_query_boolean exercises a query boolean
// parameter with a default value. Also covers the `bool` -> `boolean`
// type lowering quirk.
package routes_params_query_boolean

// ListItems swagger:route GET /items items listItemsBool
//
// List items, optionally including archived.
//
// Parameters:
//   + name: includeArchived
//     in: query
//     description: include archived items
//     type: bool
//     default: false
//
// Responses:
//
//	200: description: OK
func ListItems() {}
