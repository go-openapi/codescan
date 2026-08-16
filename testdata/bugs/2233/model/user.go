// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package model

// User is a model.
//
// swagger:model
type User struct {
	ID string `json:"id"`
}

// swaggUserInfo is the response wrapper.
//
// swagger:response userResponse
type swaggUserInfo struct {
	// in:body
	Body User
}
