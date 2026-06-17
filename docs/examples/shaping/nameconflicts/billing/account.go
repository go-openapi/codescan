// SPDX-License-Identifier: Apache-2.0

// Package billing is one of two packages in the name-conflicts witness that
// declare a type called Account. The scanner keys each definition by a
// compiler-unique identity, so this Account never merges with identity.Account.
package billing

// snippet:billing

// Account is the billing view of a customer account.
//
// swagger:model Account
type Account struct {
	// the current balance, in minor units
	Balance int64 `json:"balance"`
	// the ISO-4217 currency code
	Currency string `json:"currency"`
}

// endsnippet:billing
