// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2651 reproduces go-swagger issue #2651: an operation that mixes
// inline `swagger:operation` parameters with a `swagger:parameters`-bound
// struct mis-binds — the bound body schema is attached to the inline path
// parameter instead of becoming its own body parameter.
package bug2651

// swagger:model UpdateUser
type UpdateUser struct {
	FirstName string `json:"firstName"`
}

// swagger:parameters userUpdate
type swaggUserUpdateReq struct {
	// in: body
	Body UpdateUser
}

// swagger:operation PATCH /v1/users/{id} users userUpdate
//
// ---
// summary: Updates user
// parameters:
// - name: id
//   in: path
//   type: integer
//   required: true
// responses:
//   "200":
//     description: ok
func userUpdate() {}
