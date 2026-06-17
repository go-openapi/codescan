// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package responseref reproduces go-swagger issue #1109 using the OP's
// exact "200: order" form, where "order" names a model (not a named
// response object). Historically this produced an invalid spec with a
// dangling "$ref: #/responses/order". Today, scanned with -m (ScanModels),
// codescan's definition-fallback promotes the model name to a body ref
// ("$ref: #/definitions/order"); without -m the unresolved ref is dropped
// with a warning rather than emitting an invalid reference.
package responseref

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
//	200: order
func UpdateOrder() {}
