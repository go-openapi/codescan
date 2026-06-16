// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2184

// SpecialNumber is a struct overridden to int64 via swagger:type (#2184).
//
// swagger:type int64
type SpecialNumber struct {
	Numb  string
	Valid bool
}

// swagger:parameters objectGet
type objectGetParams struct {
	// ID
	// in: path
	// required: true
	ID SpecialNumber `json:"id"`
}

// swagger:route GET /object/{id} object objectGet
//
// Get object.
//
// responses:
//
//	200: description: ok
func ObjectGet() {}
