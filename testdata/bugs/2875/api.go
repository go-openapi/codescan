// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2875

// swagger:model SimpleOne
type SimpleOne struct {
	ID int64 `json:"id"`
}

// swagger:model AllOfModel
type AllOfModel struct {
	// swagger:allOf
	SimpleOne
}
