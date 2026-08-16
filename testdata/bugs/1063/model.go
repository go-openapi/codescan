// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1063

// User has a read-only field that should appear in GET responses but be marked
// readOnly (clients/codegen exclude it from request bodies).
//
// swagger:model
type User struct {
	// read only: true
	CompanyID int64 `json:"companyID"`

	Firstname string `json:"firstname"`
}
