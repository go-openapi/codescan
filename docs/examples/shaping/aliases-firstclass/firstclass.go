// SPDX-License-Identifier: Apache-2.0

// Package firstclass holds the annotated declarations used by the "first-class
// alias" half of the "Alias rendering" how-to — an alias that keeps its own
// identity in the spec, and the two options that shape it.
//
// firstclass_test.go scans this package three times, once per alias mode, and
// writes one golden per mode, so the guide compares real output rather than
// describing it.
package firstclass

// snippet:firstclass

// Amount is the underlying model.
//
// swagger:model
type Amount struct {
	// Cents is the amount in cents.
	Cents int64 `json:"cents"`

	// Currency is the ISO currency code.
	Currency string `json:"currency"`
}

// Fee is a FIRST-CLASS alias: the swagger:model annotation keeps the alias name
// in the spec instead of dissolving it to Amount.
//
// swagger:model
type Fee = Amount

// Receipt references the alias, not the target.
//
// swagger:model
type Receipt struct {
	// Charge is the fee charged.
	Charge Fee `json:"charge"`
}

// endsnippet:firstclass
