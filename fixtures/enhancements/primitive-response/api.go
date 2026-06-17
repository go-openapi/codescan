// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package primitiveresponse exercises primitive response bodies declared via
// swagger:route response lines and swagger:response Body fields: bare and
// tagged primitives, arrays of primitives, the swagger:response workaround,
// and a model-ref control that must still resolve to a $ref (no regression).
package primitiveresponse

// swagger:route GET /tagged misc getTagged
//
// responses:
//
//	200: body:integer
func getTagged() {}

// swagger:route GET /array misc getArray
//
// responses:
//
//	200: body:[]string
func getArray() {}

// swagger:route GET /thing misc getThing
//
// responses:
//
//	200: body:Thing
func getThing() {}

// Thing is the model-ref control.
//
// swagger:model Thing
type Thing struct {
	Name string `json:"name"`
}

// arrayBodyResponse is a swagger:response whose body is an array of a
// primitive type.
//
// swagger:response arrayBodyResponse
type arrayBodyResponse struct {
	// in: body
	Body []string
}

// swagger:route GET /named misc getNamed
//
// responses:
//
//	200: arrayBodyResponse
func getNamed() {}

// getBare uses the unsupported bare/untagged primitive form. The token is read
// as a response name, found in neither responses nor definitions, so the
// response is dropped with a diagnostic pointing at the `body:` form.
//
// swagger:route GET /bare misc getBare
//
// responses:
//
//	200: string
func getBare() {}

// getReserved uses a reserved type keyword (`file`) after body:. It is not a
// valid response body type (only string/number/integer/boolean or []T), so the
// response is dropped with a diagnostic. `array`/`object`/`null` behave the
// same way.
//
// swagger:route GET /reserved misc getReserved
//
// responses:
//
//	200: body:file
func getReserved() {}
