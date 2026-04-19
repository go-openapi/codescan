// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package swagger_type_array exercises the schemaBuilder fallthrough path
// added by #11: when swagger:type is set to a value not recognised by
// swaggerSchemaForType (e.g. "array"), the builder falls through to the
// underlying type resolution rather than silently emitting an empty schema.
package swagger_type_array

// NamedStringSlice is a named slice type carrying swagger:type array.
// With the fix it expands to {type: "array", items: {type: "string"}} via
// buildNamedSlice, preserving the field description on the enclosing
// struct instead of dropping it behind a $ref.
//
// swagger:type array
type NamedStringSlice []string

// NamedFixedArray is a named fixed-length array type carrying swagger:type
// array. The fix makes buildNamedArray inline it rather than fall through
// to $ref.
//
// swagger:type array
type NamedFixedArray [5]string

// StructWithBadType is a struct whose swagger:type is set to an
// unrecognised value. The fix ensures buildNamedStruct falls through to
// makeRef so the property is still serialisable — the key assertion is
// that the referenced schema is not empty.
//
// swagger:type badvalue
// swagger:model structWithBadType
type StructWithBadType struct {
	Name string `json:"name"`
}

// ObjectStruct carries swagger:type object (unsupported by
// swaggerSchemaForType for structs). The fix inlines the struct as
// type:object rather than producing an empty schema.
//
// swagger:type object
// swagger:model objectStruct
type ObjectStruct struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Payload aggregates all four cases so a full scan walks every branch.
//
// swagger:model payload
type Payload struct {
	// Tags for this item.
	Tags NamedStringSlice `json:"tags"`

	// Labels for this item.
	Labels NamedFixedArray `json:"labels"`

	// The nested struct with an unsupported swagger:type.
	Nested StructWithBadType `json:"nested"`

	// Headers for this request.
	Headers ObjectStruct `json:"headers"`
}
