// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2419

import "github.com/go-openapi/codescan/fixtures/bugs/2419/sub"

// SetEmailRequest overrides an external-type field with swagger:type (#2419).
//
// swagger:model
type SetEmailRequest struct {
	// swagger:type string
	UserID *sub.StringValue `json:"user_id"`
	Email  string           `json:"email"`
}
