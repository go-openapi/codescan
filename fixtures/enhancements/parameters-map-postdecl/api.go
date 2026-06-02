// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package parameters_map_postdecl witnesses a bug in the parameters
// builder's buildFromFieldMap path.
//
// LocalItem below is deliberately NOT annotated with swagger:model. Its
// only reachable site is the map-valued body parameter on MapParams.
// The parameters builder walks into buildFromFieldMap, which spins up a
// fresh schema sub-builder for the map's value type. The sub-builder
// discovers LocalItem and registers it on its own PostDeclarations
// slice. The parameters builder is supposed to propagate that
// registration to its own AppendPostDecl chain — but
// buildFromFieldMap omits that loop (every other buildFromFieldXxx
// method has it).
//
// Without the propagation, spec.Builder.buildDiscovered never sees
// LocalItem; the resulting definitions section is missing the schema,
// and the spec is internally inconsistent (the map's value shape
// vanishes silently).
//
// The pre-fix golden captures this buggy state. The fix commit
// regenerates the golden to show LocalItem appearing in the
// definitions section.
package parameters_map_postdecl

// LocalItem — NOT annotated; reachable only via the map field on
// MapParams below.
type LocalItem struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

// MapParams sends a map body parameter keyed by id with LocalItem values.
//
// swagger:parameters mapBody
type MapParams struct {
	// Items is a body parameter of type map[string]LocalItem.
	//
	// in: body
	// required: true
	Items map[string]LocalItem `json:"items"`
}

// swagger:operation POST /items mapBody
//
// Send a map body.
//
// ---
// responses:
//   "200":
//     description: OK
func _() {}
