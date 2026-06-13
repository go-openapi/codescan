// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2961

// swagger:model Sample
type Sample struct {
	// enum: 1,2,3
	Example uint `json:"example"`

	// enum: 10,20,30
	Example2 uint64 `json:"example2"`
}
