// SPDX-License-Identifier: Apache-2.0

// Package identity is the second package in the name-conflicts witness that
// declares a type called Account. It carries entirely different fields from
// billing.Account; the two must stay distinct definitions.
package identity

// snippet:identity

// Account is the identity view of a customer account.
//
// swagger:model Account
type Account struct {
	// the login email
	Email string `json:"email"`
	// whether the email has been verified
	Verified bool `json:"verified"`
}

// endsnippet:identity
