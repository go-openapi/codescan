// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug3117

// Payload is the body model.
//
// swagger:model
type Payload struct {
	Name string `json:"name"`
}

// swagger:parameters createIt
type createParams struct {
	// in: body
	Body Payload
}

// swagger:route POST /it it createIt
//
// Create it.
//
// responses:
//
//	200: description: ok
func CreateIt() {}
