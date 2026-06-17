// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2106

// User probes field-level Extensions on a NON-array vs array field (#2106).
//
// swagger:model
type User struct {
	// the name
	//
	// Extensions:
	// ---
	// x-scalar-ext: scalarvalue
	// ---
	Name string `json:"name"`

	// the friends
	//
	// Extensions:
	// ---
	// x-array-ext: arrayvalue
	// ---
	Friends []string `json:"friends"`
}
