// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2317

// Friend is referenced via pointer.
//
// swagger:model
type Friend struct {
	Name string `json:"name"`
}

// Person has a pointer field with an x-nullable Extensions block (#2317).
//
// swagger:model
type Person struct {
	// Extensions:
	// ---
	// x-nullable: true
	// ---
	Friend *Friend `json:"friend"`
}
