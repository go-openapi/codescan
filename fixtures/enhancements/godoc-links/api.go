// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package godoclinks exercises the CleanGoDoc godoc-syntax filter (P1: bracket
// stripping, identifier humanization, reference-definition-line dropping, and
// sentence-initial titleizing). It is scanned both with the option off (godoc
// emitted verbatim) and on.
package godoclinks

// Widget is owned by a [Person] and tracked in a [inventory.Ledger].
//
// More detail references the [Person] again and a [*Widget] pointer.
//
// [the spec]: https://example.com/spec
//
// swagger:model
type Widget struct {
	// Holder references the [Person] who owns this widget.
	Holder string `json:"holder"`

	// Index is element [0] in the [see notes] list; the [id] stays bare.
	Index int `json:"index"`
}

// Person represents an individual.
//
// swagger:model
type Person struct {
	// CustName mirrors the [Order.CustName] doc-link convention.
	CustName string `json:"custName"`
}
