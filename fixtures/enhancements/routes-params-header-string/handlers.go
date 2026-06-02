// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_header_string exercises a header string
// parameter with pattern and format.
package routes_params_header_string

// SecureItems swagger:route GET /items items secureItems
//
// List items with auth header.
//
// Parameters:
//   + name: X-Request-ID
//     in: header
//     description: request correlation id
//     type: string
//     pattern: ^[0-9a-fA-F-]+$
//     format: uuid
//     required: true
//
// Responses:
//
//	200: description: OK
func SecureItems() {}
