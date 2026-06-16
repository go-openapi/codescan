// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1402

// Bag has a map[string]interface{} field (#1402).
//
// swagger:model
type Bag struct {
	Data map[string]interface{} `json:"data"`
}
