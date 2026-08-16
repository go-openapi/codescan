// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_positional exercises the untagged-first-token
// response form: `200: someResponse` is treated as response="someResponse".
package routes_responses_positional

// SomeResponse is a top-level response object.
//
// swagger:response someResponse
type SomeResponse struct {
	// in: body
	Body struct {
		Status string `json:"status"`
	}
}

// ListItems swagger:route GET /items items listItemsPositional
//
// List items.
//
// Responses:
//
//	200: someResponse
func ListItems() {}
