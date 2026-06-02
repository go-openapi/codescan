// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package enum_overrides isolates the v1 behavior of `enum:` when it
// coexists with (or replaces) `swagger:enum TypeName` const-value
// inference. The golden output of TestCoverage_EnumOverrides is the
// factual reference for what the v2 parser migration must preserve
// — or consciously diverge from — under W2's override semantics
// (`.claude/plans/workshops/w2-enum.md` §2.6).
//
// Five cases, one per model in this file:
//
//	A. swagger:enum + matching consts, no inline enum on field → consts
//	B. inline comma-list on field, no swagger:enum, no consts       → inline
//	C. inline JSON array on field, no swagger:enum, no consts       → inline
//	D. swagger:enum but NO matching consts in package               → ?
//	E. swagger:enum + matching consts + inline enum on field        → ?
package enum_overrides

// --- Case A: swagger:enum + matching consts ---

// PriorityA is a classic linked-const enum.
//
// swagger:enum PriorityA
type PriorityA string

const (
	PriorityALow  PriorityA = "low"
	PriorityAMed  PriorityA = "medium"
	PriorityAHigh PriorityA = "high"
)

// NotificationA exercises case A: field uses PriorityA, no inline
// enum override.
//
// swagger:model NotificationA
type NotificationA struct {
	// required: true
	ID int64 `json:"id"`

	// The priority level. Enum values come from PriorityA's consts.
	Priority PriorityA `json:"priority"`
}

// --- Case B: inline comma-list on field, no swagger:enum ---

// NotificationB exercises case B: plain string field with inline
// comma-list enum. No swagger:enum on the type, no consts in code.
//
// swagger:model NotificationB
type NotificationB struct {
	// The priority level.
	//
	// enum: low, medium, high
	Priority string `json:"priority"`
}

// --- Case C: inline JSON-array on field, no swagger:enum ---

// NotificationC exercises case C: inline JSON-array enum.
//
// swagger:model NotificationC
type NotificationC struct {
	// The priority level.
	//
	// enum: ["low","medium","high"]
	Priority string `json:"priority"`
}

// --- Case D: swagger:enum with no matching consts ---

// PriorityD has a swagger:enum annotation but no corresponding
// const declarations in this package. The builder's FindEnumValues
// call returns an empty slice; the test captures how the spec
// renders in that case.
//
// swagger:enum PriorityD
type PriorityD string

// NotificationD exercises case D.
//
// swagger:model NotificationD
type NotificationD struct {
	// The priority level.
	Priority PriorityD `json:"priority"`
}

// --- Case E: swagger:enum + matching consts + inline override ---

// PriorityE has both a linked-const set AND fields will provide an
// inline override.
//
// swagger:enum PriorityE
type PriorityE string

const (
	PriorityELow  PriorityE = "low"
	PriorityEMed  PriorityE = "medium"
	PriorityEHigh PriorityE = "high"
)

// NotificationE exercises case E: the inline enum on the field
// competes with the const-derived enum from PriorityE. The golden
// output captures which one wins in v1.
//
// swagger:model NotificationE
type NotificationE struct {
	// Inline enum provides a narrower set than the const block.
	//
	// enum: urgent, normal
	Priority PriorityE `json:"priority"`
}
