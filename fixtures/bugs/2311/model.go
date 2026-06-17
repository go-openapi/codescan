// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2311

// Account probes swagger:ignore on a struct field (#2311).
//
// swagger:model
type Account struct {
	Name string `json:"name"`
	// swagger:ignore
	Secret string `json:"secret"`
}
