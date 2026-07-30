// SPDX-License-Identifier: Apache-2.0

// Package omit holds the annotated declarations used by the `swagger:omit`
// reference page and the "Hiding promoted fields" how-to. omit_test.go scans it
// and writes the golden fragments both pages render, so the documentation can
// never drift from the scanner's real output.
package omit

import "time"

// snippet:shared

// User is the shared domain type — deliberately free of any swagger annotation.
// The point of the idiom is that you do not have to touch it.
type User struct {
	ID      int64
	Name    string
	Created time.Time
}

// endsnippet:shared

// snippet:omit

// CreateUserParams is the request body: the same User, minus the fields the
// server assigns. `swagger:omit` sits on the embed, so the targets are plain
// field names of the embedded type.
//
// swagger:parameters createUser
type CreateUserParams struct {
	// in: body
	Body struct {
		// swagger:omit ID,Created
		User
	}
}

// endsnippet:omit

// UserResponse returns the whole type — nothing is omitted here, so the response
// documents every field.
//
// swagger:response userResponse
type UserResponse struct {
	// in: body
	Body User
}

// swagger:route POST /users users createUser
//
// Creates a user.
//
// responses:
//
//	200: userResponse
