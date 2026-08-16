// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_query_array exercises a query array parameter
// with enum + minlength/maxlength validations.
package routes_params_query_array

// FilterByTags swagger:route GET /items items filterByTags
//
// Filter items by tags.
//
// Parameters:
//   + name: tags
//     in: query
//     description: tags filter
//     type: array
//     enum: red,green,blue
//     minlength: 1
//     maxlength: 5
//
// Responses:
//
//	200: description: OK
func FilterByTags() {}
