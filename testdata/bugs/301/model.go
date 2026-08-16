// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug301

import "github.com/go-openapi/strfmt"

// User represents the user for this application
//
// A user is the security principal for this application.
// It's also used as one of main axis for reporting.
//
// A user can have friends with whom they can share what they like.
//
// swagger:model
type User struct {
	// the id for this user
	//
	// required: true
	// minimum: 1
	ID int64 `json:"id"`

	// the name for this user
	// required: true
	// min length: 3
	Name string `json:"name"`

	// the email address for this user
	//
	// required: true
	Email strfmt.Email `json:"login"`

	// the friends for this user
	Friends []User `json:"friends"`
}
