// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2479

// CreateUser opts OUT of global security with an empty Security requirement
// (go-swagger#2479). `Security: []` must emit an explicit `security: []` on the
// operation, overriding any global security.
//
// swagger:route POST /users user createUser
//
// Creates a new user.
//
// Security: []
//
// responses:
//
//	200: description: ok
func CreateUser() {}
