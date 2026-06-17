// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug958 reproduces go-swagger issue #958 ("how to specify sample
// values in user structs"): a field-level `default:` / `example:` was
// carried for a plain builtin field but dropped when the field used a named
// (defined) type such as `type SpecificType string`.
//
// The expected behaviour — locked by the golden — is that both default and
// example survive for defined-type fields. Because a $ref cannot carry
// sibling keywords, they ride the override arm of an allOf compound (the
// same draft-4-compliant shape used for description/example on $ref'd
// fields; see fixtures/bugs/3125).
package bug958

// PlainStruct uses a plain string field carrying both default and example.
//
// swagger:model PlainStruct
type PlainStruct struct {
	// Sample value in the struct
	// required: true
	// default: samplesample
	// example: examplevalue
	Value string `json:"value"`
}

type SpecificType string

// DefinedStruct uses a named (defined) type field carrying both default and example.
//
// swagger:model DefinedStruct
type DefinedStruct struct {
	// Sample value in the struct
	// required: true
	// default: samplesample
	// example: examplevalue
	Value SpecificType `json:"value"`
}
