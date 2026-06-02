// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_empty_value exercises an empty response
// value: `204:` with nothing after the colon — legacy produces an
// empty Response{} under the code.
package routes_responses_empty_value

// DeleteItem swagger:route DELETE /items/{id} items deleteItemEmpty
//
// Delete an item.
//
// Responses:
//
//	204:
//	404: description: not found
func DeleteItem() {}
