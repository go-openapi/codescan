// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3005 reproduces go-swagger issue #3005 ("additionalProperties are
// lost when generating spec from code"): a map field intended to carry the
// model's additionalProperties (excluded from named properties via json:"-")
// is dropped, so no additionalProperties is emitted on the object schema.
package bug3005

// swagger:model TestAdditionalProperties
type TestAdditionalProperties struct {
	// field1
	Field1 string `json:"field1,omitempty"`

	// test additional properties
	TestAdditionalProperties map[string]float64 `json:"-"`
}
