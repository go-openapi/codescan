// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package apfield exercises the field-level additionalProperties: keyword:
// overriding a map element schema, true/typed/model-reference values, the
// allOf-sibling form on a referenced-model field, and the lowest-priority
// contradiction on a non-object field.
package apfield

// Inner is referenced as a field-level additionalProperties value.
//
// swagger:model Inner
type Inner struct {
	Name string `json:"name"`
}

// Holder carries fields decorated with the additionalProperties: keyword.
//
// swagger:model Holder
type Holder struct {
	// OverriddenMap's element schema (string) is overridden to integer.
	//
	// additionalProperties: integer
	OverriddenMap map[string]string `json:"overriddenMap"`

	// AnyMap allows any extra value.
	//
	// additionalProperties: true
	AnyMap map[string]string `json:"anyMap"`

	// RefMap's values reference a model.
	//
	// additionalProperties: Inner
	RefMap map[string]string `json:"refMap"`

	// ClosedInner references a model and forbids extra keys via an allOf
	// sibling (the $ref is preserved).
	//
	// additionalProperties: false
	ClosedInner Inner `json:"closedInner"`

	// BadField is a string; additionalProperties is the lowest-priority
	// annotation, so it is dropped with a diagnostic.
	//
	// additionalProperties: true
	BadField string `json:"badField"`
}
