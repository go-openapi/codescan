// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package godoclinks exercises the CleanGoDoc godoc-syntax filter. It is scanned
// both with the option off (godoc emitted verbatim) and on. With the option on,
// doc-links that resolve to a scanned model are recomposed to that model's
// EXPOSED name (Widget -> "gizmo" via swagger:model, Order.CustName ->
// "Order.customer_name" via the json tag); unresolved links are humanized;
// reference-definition lines are dropped; and non-doc-link brackets are left
// intact.
package godoclinks

import "github.com/go-openapi/codescan/fixtures/enhancements/godoc-links/inventory"

// Widget is the primary [Gadget] holder and references [Order.CustName].
//
// More detail mentions a [*Gadget] pointer, a [inventory.Ledger], and an
// unknown [Sprocket].
//
// [the spec]: https://example.com/spec
//
// swagger:model gizmo
type Widget struct {
	// Holder points at the [Gadget] that owns this widget.
	Holder string `json:"holder"`

	// Ledger is the cross-package [inventory.Ledger] reference.
	Ledger *inventory.Ledger `json:"ledger"`

	// Index is element [0] in the [see notes] list; the [id] stays bare.
	Index int `json:"index"`
}

// Gadget is a small component.
//
// swagger:model
type Gadget struct {
	Name string `json:"name"`
}

// Order captures a purchase.
//
// swagger:model
type Order struct {
	// CustName is the buyer.
	CustName string `json:"customer_name"`
}
