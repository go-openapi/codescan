// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_query_number exercises a query number parameter
// with min/max/default landed via legacy short-form keys.
package routes_params_query_number

// FilterItems swagger:route GET /items items filterItems
//
// Filter items by price.
//
// Parameters:
//   + name: price
//     in: query
//     description: maximum price
//     type: number
//     min: 0.0
//     max: 999.99
//     default: 100.0
//
// Responses:
//
//	200: description: OK
func FilterItems() {}
