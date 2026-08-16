// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2396

// Bracket uses the bracketed `enum: [a, b, c]` form (go-swagger#2396). The
// surrounding brackets must be stripped so the enum is [issues, pulls, projects]
// — not ["[issues", "pulls", "projects]"].
//
// swagger:model
type Bracket struct {
	// kind
	// enum: [issues, pulls, projects]
	Kind string `json:"kind"`
}

// Spaced uses the unbracketed `enum: a, b, c` form (green guard rail — already
// trims correctly).
//
// swagger:model
type Spaced struct {
	// kind
	// enum: issues, pulls, projects
	Kind string `json:"kind"`
}
