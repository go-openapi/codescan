// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2064

// swagger:parameters createThing
type thingParams struct {
	// desc
	// in: body
	// example: example
	// default: def
	Body string
}

// swagger:route POST /thing thing createThing
//
// Create.
//
// responses:
//
//	200: description: ok
func CreateThing() {}
