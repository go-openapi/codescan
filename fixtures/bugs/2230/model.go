// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2230

import "encoding/json"

// SearchObject has a json.RawMessage field (#2230).
//
// swagger:model
type SearchObject struct {
	// the limit
	// example: 200
	Limit int `json:"limit"`
	// the search payload
	Search json.RawMessage `json:"search"`
}
