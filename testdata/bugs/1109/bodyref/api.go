// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bodyref reproduces go-swagger issue #1109 using casualjim's
// recommended "200: body:order" form, which references a model as the
// response body. Scanned with -m, codescan emits a 200 response with
// "schema: $ref #/definitions/order" and the order definition.
package bodyref

// An Order for one or more pets by a user.
// swagger:model order
type Order struct {
	// required: true
	ID int64 `json:"id"`
}

// UpdateOrder swagger:route PUT /orders/{id} orders updateOrder
//
// Updates an order.
//
// Responses:
//
//	200: body:order
func UpdateOrder() {}
