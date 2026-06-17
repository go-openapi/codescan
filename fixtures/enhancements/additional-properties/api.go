// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package additionalprops exercises the type-level swagger:additionalProperties
// marker: true / false / a typed value / a model reference, the map-override
// case, coexistence with other object validations, and the lowest-priority
// precedence rule (the marker only rides on an object).
package additionalprops

// Thing is referenced as an additionalProperties value type.
//
// swagger:model Thing
type Thing struct {
	Name string `json:"name"`
}

// ForbidExtras is a closed object: named properties only, no extra keys.
//
// swagger:model ForbidExtras
// swagger:additionalProperties false
type ForbidExtras struct {
	A string `json:"a"`
	B int    `json:"b"`
}

// AllowExtras is an open object: named properties plus arbitrary extra keys.
//
// swagger:model AllowExtras
// swagger:additionalProperties true
type AllowExtras struct {
	A string `json:"a"`
}

// TypedExtras complements its named property with typed (integer) values.
//
// swagger:model TypedExtras
// swagger:additionalProperties integer
type TypedExtras struct {
	A string `json:"a"`
}

// RefExtras references a model as its additionalProperties value schema.
//
// swagger:model RefExtras
// swagger:additionalProperties Thing
type RefExtras struct {
	A string `json:"a"`
}

// OverrideMap is a map whose element schema (string) is overridden by the
// marker to integer additionalProperties.
//
// swagger:model OverrideMap
// swagger:additionalProperties integer
type OverrideMap map[string]string

// BoundedExtras shows additionalProperties coexisting with maxProperties.
//
// maxProperties: 10
//
// swagger:model BoundedExtras
// swagger:additionalProperties true
type BoundedExtras struct {
	A string `json:"a"`
}

// Contradiction resolves to a string via swagger:type; additionalProperties is
// the lowest-priority annotation, so it is dropped with a diagnostic.
//
// swagger:model Contradiction
// swagger:type string
// swagger:additionalProperties true
type Contradiction struct {
	A string `json:"a"`
}
