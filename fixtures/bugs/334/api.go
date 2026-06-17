// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug334

// swagger:model
// User represents the user for this application
//
// A user is the security principal for this application.
type User struct {
	// required: true
	// minimum: 1
	ID int64 `json:"id"`

	// required: true
	// min length: 3
	Firstname string `json:"firstname"`
}

// UsersResponse is the list response.
//
// swagger:response usersResponse
type UsersResponse struct {
	// in: body
	Body []User
}

// GetUsers mirrors the reporter's layout: the swagger:route annotation lives
// INSIDE the function body rather than in the function's doc comment.
func GetUsers() {
	// swagger:route GET /users listUsers users
	//
	// Lists users.
	//
	//     Responses:
	//       200: usersResponse
	_ = 0
}
