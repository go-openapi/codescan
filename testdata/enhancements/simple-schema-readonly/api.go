// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package simple_schema_readonly exercises the schema builder's
// SimpleSchema-mode gate on the full-Schema-only `readOnly:`
// keyword. The fixture writes `readOnly: true` on a struct property
// reached during a non-body parameter resolution; the schema
// builder's Bool dispatcher emits CodeUnsupportedInSimpleSchema and
// skips the write.
//
// The Filter field must be an **anonymous** struct so the schema
// builder walks it inline rather than emitting a $ref (named
// structs short-circuit via resolveRefOr in buildNamedType and
// never reach walkSchemaLevel under SimpleSchema mode).
package simple_schema_readonly

// ReadOnlyParams demonstrates a non-body parameter typed as an
// anonymous struct, triggering the SimpleSchema-mode descent
// through the schema builder's struct walker.
//
// swagger:parameters readOnlyOp
type ReadOnlyParams struct {
	// Filter is the offending parameter — typed as an anonymous
	// struct under `in: query` is itself outside SimpleSchema's
	// allowed type set, but the schema builder walks it inline
	// (rather than $ref'ing it) and so reaches the inner
	// `readOnly:` annotation. The gate emits
	// CodeUnsupportedInSimpleSchema and skips the write.
	//
	// in: query
	Filter struct {
		// Hidden is meant to be read-only — but `readOnly:` is a
		// full-Schema-only keyword and cannot appear under
		// SimpleSchema mode.
		//
		// readOnly: true
		Hidden bool `json:"hidden"`
	} `json:"filter"`
}

// DoReadOnly handles the violating route.
//
// swagger:route GET /readonly ro readOnlyOp
//
// Responses:
//
//	200: description: OK
func DoReadOnly() {}
