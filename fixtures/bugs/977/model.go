// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug977

// Role is an aliased string type used as a map key (#977).
type Role string

// Acl maps an aliased-string key to a value.
//
// swagger:model
type Acl struct {
	Perms map[Role][]int `json:"perms"`
}
