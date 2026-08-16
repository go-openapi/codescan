// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2575

// swagger:parameters getPet
type getPetParams struct {
	// X-Request-ID custom header.
	// in: header
	// required: true
	XRequestID string `json:"X-Request-Id"`
}

// swagger:route GET /pets pets getPet
//
// List all pets.
//
// responses:
//
//	200: description: ok
func GetPet() {}
