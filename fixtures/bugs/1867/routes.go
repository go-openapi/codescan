// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1867

// swagger:parameters patchOpenHybridRoute
type patchBody struct {
	// in: body
	// required: true
	Body struct {
		EndTime string `json:"end_time"`
	}
}

// PatchViaRoute documents a PATCH with a body parameter via swagger:route.
//
// swagger:route PATCH /hybrid/route hybrid patchOpenHybridRoute
//
// Patch via route.
//
// Responses:
//
//	200: description: ok
func PatchViaRoute() {}

// swagger:operation PATCH /hybrid/op hybrid patchOpenHybridOp
//
// ---
// summary: Patch via operation.
// responses:
//
//	'200':
//	  description: ok
func PatchViaOperation() {}
