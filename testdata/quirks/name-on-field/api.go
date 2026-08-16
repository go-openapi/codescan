// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package nameonfield is the F5 witness: swagger:name on a struct field must
// override the emitted property name, matching its documented "field OR
// method" scope and the behaviour already honoured on interface methods.
package nameonfield

// Account renames fields via swagger:name.
//
// swagger:model Account
type Account struct {
	// Balance is renamed via swagger:name with no json tag present.
	//
	// swagger:name balance
	Balance int64

	// Currency is renamed via swagger:name, overriding the json tag.
	//
	// swagger:name currencyCode
	Currency string `json:"currency"`

	// Plain has no override and keeps its json tag name.
	Plain string `json:"plain"`
}
