// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug622

// swagger:response personResponse
type PersonResponse struct {
	// in: body
	Body struct {
		// The person's name
		// required: true
		// example: Bob
		Name string `json:"name"`
	}
}
