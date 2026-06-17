// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2286

// LogoutResponse is used directly as a response (#2286).
//
// swagger:model logoutResponse
type LogoutResponse struct {
	// in:body
	Payload string `json:"body,omitempty"`
}

// swagger:route GET /logout auth authLogout
//
// Logout.
//
// responses:
//
//	200: logoutResponse
func AuthLogout() {}
