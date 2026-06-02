// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_query_string exercises a query string parameter
// with pattern and format validations.
package routes_params_query_string

// SearchItems swagger:route GET /items items searchItems
//
// Search items by tag.
//
// Parameters:
//   + name: tag
//     in: query
//     description: tag to filter by
//     type: string
//     pattern: ^[a-z]+$
//     format: byte
//
// Responses:
//
//	200: description: OK
func SearchItems() {}
