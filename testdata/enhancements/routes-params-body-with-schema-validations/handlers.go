// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_params_body_with_schema_validations exercises the
// "force-the-spec" case: a body ref to a known model plus inline
// schema validations that land on param.Schema.{Minimum,Maximum,...}
// per legacy processSchema behavior.
package routes_params_body_with_schema_validations

// Pet is a pet on offer.
//
// swagger:model
type Pet struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// CreatePet swagger:route POST /pets pets createPetWithOverrides
//
// Create a pet with author-asserted schema constraints.
//
// Parameters:
//   + name: body
//     in: body
//     description: pet to create
//     required: true
//     type: Pet
//     min: 0
//     max: 999
//     format: special
//
// Responses:
//
//	201: description: created
func CreatePet() {}
