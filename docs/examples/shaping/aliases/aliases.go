// SPDX-License-Identifier: Apache-2.0

// Package aliases holds the annotated declarations used by the "Alias rendering"
// how-to. aliases_test.go scans it and writes the golden fragment the guide
// renders.
//
// This example uses an unannotated alias (the dissolve case), the common
// behaviour the guide leads with; the first-class-alias modes are described
// conceptually there.
//
// (Historical note: those modes once hung the scanner — quirk F9 — which is why
// this example avoids them. The hang was fixed and is locked across all three
// alias modes by TestQuirk_AliasModelNoHang.)
package aliases

// snippet:alias

// Money is the underlying model.
//
// swagger:model
type Money struct {
	// Cents is the amount in cents.
	Cents int64 `json:"cents"`

	// Currency is the ISO currency code.
	Currency string `json:"currency"`
}

// Price is a Go alias of Money. By default an alias is a Go implementation
// detail: at use sites it dissolves to its target, producing no definition of
// its own.
type Price = Money

// Invoice references Price; the field resolves to Money.
//
// swagger:model
type Invoice struct {
	// Total is the invoice total.
	Total Price `json:"total"`
}

// endsnippet:alias
