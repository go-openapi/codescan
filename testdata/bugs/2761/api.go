// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2761

// swagger:model BaseResponse
type BaseResponse struct {
	Error string `json:"error,omitempty"`
	OK    bool   `json:"ok"`
}

// swagger:model User
type User struct {
	Name string `json:"name"`
}

// swagger:response userResponse
type UserResponse struct {
	// in: body
	Body struct {
		// swagger:allOf
		BaseResponse

		User User `json:"user"`
	}
}
