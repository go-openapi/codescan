// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2663

// CPF is an in-house codified username (a named string type).
type CPF string

// AppUserCredentials is the JSON request body for a login.
//
// swagger:model
type AppUserCredentials struct {
	// The user's codified username.
	Username CPF `json:"username"`
	// The user's password.
	Password string `json:"password"`
}

// The recommended idiom: ONE field carries `in: body`; its Go type is the
// body schema. A named type yields a reusable $ref definition.
//
// swagger:parameters login
type loginParams struct {
	// The login credentials.
	// in: body
	// required: true
	Body AppUserCredentials `json:"body"`
}

// swagger:route POST /login auth login
//
// Authenticate a user.
//
// responses:
//
//	200: description: ok
func Login() {}
