// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1117

// Item is one array element.
//
// swagger:model
type Item struct {
	ID int64 `json:"id"`
}

// ArrayResponse is a response whose body is an array type.
//
// swagger:response arrayResponse
type ArrayResponse struct {
	// in: body
	Body []Item
}
