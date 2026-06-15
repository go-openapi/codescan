// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1992

// User has server-assigned fields that the reporter wanted to hide from create
// requests (go-swagger#1992). The OAS2 idiom is `readOnly: true` — codescan
// emits one schema per type, so per-operation field hiding is out of scope, but
// readOnly marks the server-owned fields.
//
// swagger:model
type User struct {
	// ID is server-assigned.
	//
	// read only: true
	ID int64 `json:"id"`
	// Name is client-supplied.
	Name string `json:"name"`
	// Created is server-assigned.
	//
	// read only: true
	Created string `json:"created"`
}
