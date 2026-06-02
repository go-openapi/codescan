// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package simple_schema_violation exercises the M1 exit validator on
// the parameter SimpleSchema path. A non-body parameter typed as a
// named string with a `swagger:type object` decl-level override
// resolves to a schema with `Type == "object"` — invalid under OAS v2
// SimpleSchema. The exit validator emits
// CodeUnsupportedInSimpleSchema and resets the target to empty `{}`.
package simple_schema_violation

// ObjectOverride is a named string carrying a decl-level
// `swagger:type object` override. The override is honoured by the
// schema builder (classifierNamedBasic arm); under SimpleSchema mode
// the exit validator catches the resulting `Type == "object"` and
// resets the parameter back to empty `{}`.
//
// swagger:type object
type ObjectOverride string

// ViolatingParams demonstrates a query parameter whose Go type
// resolves to an object-shaped schema — invalid under SimpleSchema.
//
// swagger:parameters violationOp
type ViolatingParams struct {
	// Bad is the offending parameter — its type carries a
	// decl-level override that the schema builder honours, producing
	// an object-typed SimpleSchema. The M1 exit validator emits a
	// CodeUnsupportedInSimpleSchema diagnostic and wipes the target.
	//
	// in: query
	Bad ObjectOverride `json:"bad"`
}

// DoViolation handles the violating route.
//
// swagger:route GET /violation viol violationOp
//
// Responses:
//
//	200: description: OK
func DoViolation() {}
