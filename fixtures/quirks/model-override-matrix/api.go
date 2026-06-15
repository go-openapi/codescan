// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package modeloverridematrix witnesses the CURRENT behaviour (post-F3) of
// combining swagger:model with a per-type override — swagger:strfmt (F1),
// swagger:type (F2) and swagger:enum (F4) — plus the bare swagger:enum
// const-collection case. No fix here; the golden is the baseline the F1/F2/F4
// reconciliation is measured against. See doc-site-quirks.md F1/F2/F4.
package modeloverridematrix

// UUID is a string-format type carrying BOTH swagger:strfmt and swagger:model
// (F1): does it publish a {type:string,format:uuid} definition that fields
// $ref, or inline + an orphan definition?
//
// swagger:strfmt uuid
// swagger:model UUID
type UUID string

// RawID is a byte-array type overridden to string, with swagger:model (F2).
//
// swagger:type string
// swagger:model RawID
type RawID [12]byte

// Status is an enum type (named swagger:enum) with swagger:model (F4): values
// inline on referencing fields, or a $ref to an enum definition?
//
// swagger:enum Status
// swagger:model Status
type Status string

// Status enum values.
const (
	// StatusActive is active.
	StatusActive Status = "active"
	// StatusClosed is closed.
	StatusClosed Status = "closed"
)

// Priority is an enum declared with the BARE swagger:enum (no name arg) (F4b):
// does it collect the consts below?
//
// swagger:enum
// swagger:model Priority
type Priority int

// Priority enum values.
const (
	// PriorityLow is low.
	PriorityLow Priority = 1
	// PriorityHigh is high.
	PriorityHigh Priority = 9
)

// Holder references each override type to expose field-site rendering.
//
// swagger:model Holder
type Holder struct {
	ID       UUID     `json:"id"`
	Raw      RawID    `json:"raw"`
	State    Status   `json:"state"`
	Priority Priority `json:"priority"`
}
