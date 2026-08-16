// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2218

// swagger:parameters listThings
type listParams struct {
	// the filter
	// in: query
	Filter string `json:"filter"`
}

// swagger:route GET /things things listThings
//
// List things.
//
// responses:
//
//	200: description: ok
func ListThings() {}
