// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2133

// swagger:parameters listAccounts
type listAccountsParams struct {
	// List of accounts in comma separated format.
	// in: path
	// required: true
	Accounts string `json:"accounts"`
}

// swagger:route GET /accounts/{accounts} accounts listAccounts
//
// List accounts.
//
// responses:
//
//	200: description: ok
func ListAccounts() {}
