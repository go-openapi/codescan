// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_responses_definition_fallback exercises the legacy
// silent flip from response-ref to body-ref when the untagged name is
// not in the operation's responses map but IS in definitions
// (parseResponseLine lines 83-87). Pet is a model, not a response, so
// `200: Pet` should resolve as a body ref to #/definitions/Pet.
package routes_responses_definition_fallback

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetPet swagger:route GET /pets/{id} pets getPetFallback
//
// Get a pet.
//
// Responses:
//
//	200: Pet
func GetPet() {}
