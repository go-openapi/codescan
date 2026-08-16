// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1436

// Reachable is referenced by a route (always emitted).
//
// swagger:model
type Reachable struct {
	A string `json:"a"`
}

// Standalone is a swagger:model not referenced by any route (#1436: only with -m).
//
// swagger:model
type Standalone struct {
	B string `json:"b"`
}

// swagger:response reachableResp
type reachableResp struct {
	// in: body
	Body Reachable
}

// swagger:route GET /thing thing getThing
//
// Get thing.
//
// responses:
//
//	200: reachableResp
func GetThing() {}
