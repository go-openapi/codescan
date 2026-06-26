// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters (Fixture 1) witnesses the spec-level shared
// namespace for the shared-parameters feature (go-swagger#2632): the
// wildcard target, both parameter reference channels, and a shared
// response. Each form is documented on the declaration that exercises it.
//
// See .claude/plans/features/shared-parameters-fixtures.md for the grammar
// and the expected spec this fixture should produce.
package shared_parameters

// CommonHeaders registers a reusable header parameter at the spec top
// level (#/parameters/X-Request-ID). Register-only: no operation list,
// so nothing references it directly here — it is pulled in below via the
// standalone reference channel.
//
// swagger:parameters *
type CommonHeaders struct {
	// RequestID correlates a request across services.
	//
	// in: header
	RequestID string `json:"X-Request-ID"`
}

// AuthHeader registers #/parameters/X-API-Key AND $ref's it into the
// createPet operation (the small-spec "* opID" convenience).
//
// swagger:parameters * createPet
type AuthHeader struct {
	// APIKey authorises access.
	//
	// in: header
	// required: true
	APIKey string `json:"X-API-Key"`
}

// ListPetsParams are inlined into listPets (existing per-operation form).
//
// swagger:parameters listPets
type ListPetsParams struct {
	// Limit caps the number of pets returned.
	//
	// in: query
	Limit int `json:"limit"`
}

// CreatePetParams are inlined into createPet; createPet therefore mixes
// an inlined body parameter with the $ref'd X-API-Key above.
//
// swagger:parameters createPet
type CreatePetParams struct {
	// in: body
	// required: true
	Body Pet `json:"body"`
}

// ErrorResponse is force-registered at #/responses/ErrorResponse and
// referenced by both routes' Responses blocks (emitted as a $ref).
//
// swagger:response *
type ErrorResponse struct {
	// in: body
	Body struct {
		// Code is a machine-readable error code.
		Code int `json:"code"`
		// Message is a human-readable error message.
		Message string `json:"message"`
	} `json:"body"`
}

// Pet is the body model.
//
// swagger:model
type Pet struct {
	// Name of the pet.
	Name string `json:"name"`
}

// ListPets lists pets. The standalone reference marker below pulls the
// shared X-Request-ID parameter into this operation as a $ref (the
// scaling channel: the shared struct need not list every operation).
//
// swagger:route GET /pets pets listPets
// swagger:parameters listPets X-Request-ID
// Responses:
//
//	default: ErrorResponse
func ListPets() {}

// CreatePet creates a pet. Its parameters come from AuthHeader
// (#/parameters/X-API-Key, $ref'd via "* createPet") and CreatePetParams
// (inlined body).
//
// swagger:route POST /pets pets createPet
// Responses:
//
//	default: ErrorResponse
func CreatePet() {}
