// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_tagged_response exercises the explicit
// `response:someName` tag form.
package routes_responses_tagged_response

// GenericError is the catch-all error response.
//
// swagger:response genericError
type GenericError struct {
	// in: body
	Body struct {
		Message string `json:"message"`
	}
}

// ListItems swagger:route GET /items items listItemsTaggedResp
//
// List items.
//
// Responses:
//
//	default: response:genericError
//	200: description: OK
func ListItems() {}
