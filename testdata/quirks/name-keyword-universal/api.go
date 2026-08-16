// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package namekeyworduniversal exercises doc-quirk G2: the `name:` keyword is
// the canonical field-naming keyword, honoured uniformly at every field site
// — model properties and interface methods, as well as the parameter and
// response fields where it already worked. The older name annotation is
// retained as a legacy form.
//
// Precedence (identical in every context):
//
//	name: keyword  >  name annotation  >  json tag  >  Go field name
package namekeyworduniversal

// Account exercises the name: keyword on model struct fields.
//
// swagger:model Account
type Account struct {
	// the running balance
	// name: balance
	Bal float64 `json:"bal"`

	// the keyword wins over a co-located legacy name annotation.
	// name: currencyCode
	// swagger:name legacyCurrency
	Currency string `json:"currency"`

	// the legacy name annotation is still honoured on its own.
	// swagger:name accountType
	Kind string

	// no override → the json tag is retained.
	Plain string `json:"plain"`
}

// Token exercises the name: keyword on interface-method properties.
//
// swagger:model Token
type Token interface {
	// name: issuedAt
	IssuedAt() int64

	// the legacy name annotation still works on a method too.
	// swagger:name subject
	Sub() string
}

// swagger:parameters createAccount
type createAccountParams struct {
	// regression guard: name: keeps working on a parameter field.
	// in: query
	// name: account_id
	AccountID string

	// G2 diagnostic: swagger:name is inert here; the author likely meant
	// the name: keyword. The annotation is dropped and a warning emitted,
	// so this parameter keeps its Go-derived name.
	// in: query
	// swagger:name region_code
	Region string
}

// accountCreated is a response whose header field misuses swagger:name.
//
// swagger:response accountCreated
type accountCreated struct {
	// G2 diagnostic: swagger:name is inert on a response header; the
	// canonical rename is the name: keyword.
	// swagger:name X-Account-Token
	Token string

	// in: body
	Body Account
}

// swagger:route POST /accounts accounts createAccount
//
// Create an account.
//
// responses:
//
//	200: accountCreated
func createAccount() {}
