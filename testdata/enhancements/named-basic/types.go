// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package named_basic exercises the schemaBuilder.buildNamedBasic branches
// for named basic types carrying swagger:strfmt, swagger:type and the
// deprecated swagger:default annotation.
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

// Grade is a named int tagged with the DEPRECATED swagger:default
// annotation. The annotation is an inert sink: Grade must emit exactly
// what it would without it — a plain named int, referenced by $ref from
// the field site — and the scan must raise a deprecation diagnostic.
//
// It used to claim the target without writing it, publishing a typeless
// schema for the declared type and a typeless property for every field
// referencing it.
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
