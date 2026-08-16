// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_description_only exercises response lines
// carrying only a description: tag with no body/response ref.
package routes_responses_description_only

// GetItem swagger:route GET /items/{id} items getItemDescOnly
//
// Get an item.
//
// Responses:
//
//	200: description: OK
//	404: description: not found
//	500: description: server error
func GetItem() {}
