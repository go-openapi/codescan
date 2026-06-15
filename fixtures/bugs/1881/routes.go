// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1881

// Item is an element model.
//
// swagger:model
type Item struct {
	Name string `json:"name"`
}

// ItemsResponse is a response whose body is an array of objects (the #1881
// "array-of-object response yields invalid spec" case).
//
// swagger:response itemsResponse
type ItemsResponse struct {
	// in: body
	Body []Item
}

// swagger:route GET /items items listItems
//
// List items.
//
// Responses:
//
//	200: itemsResponse
func ListItems() {}
