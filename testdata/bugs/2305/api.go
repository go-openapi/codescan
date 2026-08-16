// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2305

// swagger:parameters getItems
type getItemsParams struct {
	// Sort order
	// in: query
	// enum: asc,desc
	Sort string `json:"sort"`
}

// swagger:route GET /items items getItems
//
// List items.
//
// responses:
//
//	200: description: ok
func GetItems() {}
