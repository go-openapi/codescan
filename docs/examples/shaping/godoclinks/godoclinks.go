// SPDX-License-Identifier: Apache-2.0

// Package godoclinks is the test-backed witness for the "Cleaning godoc
// doc-links" how-to. It is scanned with CleanGoDoc off (godoc emitted
// verbatim) and on (doc-links recomposed to each schema's exposed name,
// unresolved links humanized, reference-definition lines dropped, ordinary
// brackets left intact).
package godoclinks

import "github.com/go-openapi/codescan/docs/examples/shaping/godoclinks/inventory"

// snippet:widget

// Widget is the primary [Gadget] holder and references [Order.CustName].
//
// More detail mentions a [*Gadget] pointer, a [inventory.Ledger], and an
// unknown [Sprocket].
//
// swagger:model gizmo
type Widget struct {
	// Holder points at the [Gadget] that owns this widget.
	Holder string `json:"holder"`

	// Ledger is the cross-package [inventory.Ledger] reference.
	Ledger *inventory.Ledger `json:"ledger"`

	// Index is element [0] in the [see notes] list; the [id] stays bare.
	Index int `json:"index"`

	// Spec points at [Gadget]; the reference-definition line below is godoc
	// link plumbing that carries no prose.
	//
	// [the spec]: https://example.com/spec
	Spec string `json:"spec"`
}

// endsnippet:widget

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
	CustName string `json:"customer_name"` //nolint:tagliatelle // snake_case demonstrates exposed-name recomposition
}
