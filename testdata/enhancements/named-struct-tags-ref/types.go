// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package named_struct_tags exercises schemaBuilder.buildNamedStruct
// branches where a named struct referenced as a field carries a
// swagger:strfmt or swagger:type annotation that overrides its schema.
package named_struct_tags

// PhoneNumber is a named struct annotated as a strfmt. When used as a
// field type the scanner emits {type: "string", format: "phone"} via the
// strfmt branch of buildNamedStruct.
//
// Since we have added the explicit model annotation, we expect this definition
// to bubble up and appear as a $ref in the spec.
//
// swagger:strfmt phone
// swagger:model
type PhoneNumber struct {
	CountryCode string
	Number      string
}

// LegacyCode is a named struct annotated with swagger:type so that its
// referenced schema is coerced to the declared swagger type rather than
// emitted as an object.
//
// swagger:type string
type LegacyCode struct {
	Version int
}

// Contact references both tagged struct types so the scanner walks the
// buildNamedStruct strfmt and typeName branches on distinct fields.
//
// swagger:model Contact
type Contact struct {
	// required: true
	ID int64 `json:"id"`

	Phone PhoneNumber `json:"phone"`

	Code LegacyCode `json:"code"`
}
