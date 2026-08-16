// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package default_int carries a parameter with a `default:` value that
// does not parse as the declared type. parseValueFromSchema is wired to
// run `strconv.Atoi` on int types, which will return an error that
// propagates up through setDefault.Parse.
package default_int

// BadParams has an int query parameter with a non-numeric default.
//
// swagger:parameters badDefaultInt
type BadParams struct {
	// in: query
	// default: notanumber
	Value int `json:"value"`
}

// DoBad swagger:route GET /bad-default bad badDefaultInt
//
// Responses:
//
//	200: description: OK
func DoBad() {}
