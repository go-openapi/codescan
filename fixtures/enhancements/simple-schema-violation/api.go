// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package simple_schema_violation exercises the two ways a non-body
// parameter can fail to be an OAS v2 SimpleSchema, which have
// different remedies.
//
//  1. The ANNOTATION asks for something the location cannot carry —
//     `swagger:type object` on a query parameter. `type` is mandatory
//     under SimpleSchema, so refusing the override and keeping the
//     Go-derived type leaves a valid parameter; the diagnostic names
//     the annotation. Honouring it and then wiping the result, as this
//     fixture used to witness, left the parameter untyped.
//
//  2. The GO TYPE itself is not representable. There is no override to
//     refuse and no fallback to keep, so the exit validator wipes the
//     target and says so — honest over lossy.
package simple_schema_violation

// ObjectOverride is a named string carrying a decl-level
// `swagger:type object` override — case 1. Under SimpleSchema the
// override is refused before it is applied and the parameter keeps
// `{type: string}`, the type its Go declaration gives it.
//
// swagger:type object
type ObjectOverride string

// ViolatingParams demonstrates a query parameter whose Go type
// resolves to an object-shaped schema — invalid under SimpleSchema.
//
// swagger:parameters violationOp
type ViolatingParams struct {
	// Bad carries an override the location cannot honour (case 1): the
	// annotation is ignored with a diagnostic and the Go type stands.
	//
	// in: query
	Bad ObjectOverride `json:"bad"`

	// Unrepresentable is case 2 — a struct has no SimpleSchema form and
	// there is no annotation to refuse, so the exit validator wipes it.
	//
	// in: query
	Unrepresentable struct {
		Left string `json:"left"`
	} `json:"unrepresentable"`

	// Errored is case 3 — an `error` has no meaning as a parameter, so the
	// field is dropped rather than described. A struct shared between a
	// parameter set and a response should lose it on the parameter side.
	//
	// This used to abort the whole scan; skip-with-a-diagnostic is the house
	// rule, and the sibling above already followed it.
	//
	// in: query
	Errored error `json:"errored"`
}

// DoViolation handles the violating route.
//
// swagger:route GET /violation viol violationOp
//
// Responses:
//
//	200: description: OK
func DoViolation() {}
