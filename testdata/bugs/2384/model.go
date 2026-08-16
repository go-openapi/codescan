// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2384

// Role probes a pattern containing backslash escapes (#2384).
//
// swagger:model
type Role struct {
	// the name
	// pattern: ^[^ \t\n](.*[^ \t\n])*$
	Name string `json:"name"`
}
