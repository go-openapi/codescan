// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug989

// ListResponse is the list response.
//
// swagger:response listResponse
type ListResponse struct {
	// in: body
	Body []string
}

// ErrResponse is the error response.
//
// swagger:response errResponse
type ErrResponse struct {
	// in: body
	Body struct {
		Message string `json:"message"`
	}
}

// ListUsers lists all the users (legacy swagger:route form).
//
// In a swagger:route Responses block a status code maps to a response name; an
// inline description value is not captured (the 403 below gets an empty
// description).
//
// swagger:route GET /users user listUsers
//
// List all the users.
//
//	Responses:
//	  200: listResponse
//	  401: errResponse
//	  403:
//	    description: Unauthorized
func ListUsers() {}

// ListAdmins lists admins (swagger:operation form).
//
// The full-YAML swagger:operation responses block DOES capture an inline
// description value — this is the supported way to add a description to a
// response without a dedicated response object.
//
// swagger:operation GET /admins admin listAdmins
//
// List all the admins.
//
// ---
// responses:
//   200:
//     description: OK
//   403:
//     description: Unauthorized
func ListAdmins() {}
