// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package named_basic exercises the schemaBuilder.buildNamedBasic branches
// for named basic types carrying swagger:strfmt, swagger:type and
// swagger:default annotations.
package named_basic

// Email is a named string with a swagger:strfmt tag. The scanner must
// emit {"type": "string", "format": "email"} via the strfmt branch.
//
// swagger:strfmt email
type Email string

// Colour is a named int whose representation is overridden via
// swagger:type so the scanner emits it as a string instead of an integer.
//
// swagger:type string
type Colour int

// Grade is a named int tagged with swagger:default which causes the
// scanner to emit an empty schema for the declared type.
//
// swagger:default Grade
type Grade int

// User embeds the three named basic types above so that the full scan
// walks buildNamedBasic for each field.
//
// swagger:model User
type User struct {
	// required: true
	ID int64 `json:"id"`

	Email Email `json:"email"`

	Colour Colour `json:"colour"`

	Grade Grade `json:"grade"`
}
