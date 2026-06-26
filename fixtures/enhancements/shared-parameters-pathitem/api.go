// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package shared_parameters_pathitem (Fixture 2) witnesses path-item
// level parameters for the shared-parameters feature (go-swagger#2632):
// path-item inline and reference forms, an operation override on
// (name, in), and exact-path application (no path hierarchy). Each form is
// documented on the declaration that exercises it.
//
// See .claude/plans/features/shared-parameters-fixtures.md for the grammar
// and the expected spec.
package shared_parameters_pathitem

// CommonHeaders registers #/parameters/X-Request-ID so the path-item
// reference marker below can $ref it.
//
// swagger:parameters *
type CommonHeaders struct {
	// RequestID correlates a request across services.
	//
	// in: header
	RequestID string `json:"X-Request-ID"`
}

// PetPathParams are inlined into the /pets path-item parameters and thus
// inherited by every operation under the EXACT path /pets (no hierarchy:
// /pets/{id} does not inherit them).
//
// swagger:parameters /pets
type PetPathParams struct {
	// APIKey authorises access to the pet store.
	//
	// in: header
	// required: true
	APIKey string `json:"X-API-Key"`
}

// ListPetsParams override the path-item X-API-Key on (name="X-API-Key",
// in="header"): per OAS2 the operation-level parameter wins, so listPets
// sees X-API-Key as optional even though the path-item marks it required.
// Both appear in the spec (co-presence); the operation one wins.
//
// swagger:parameters listPets
type ListPetsParams struct {
	// APIKey overrides the path-item parameter to make it optional here.
	//
	// in: header
	// required: false
	APIKey string `json:"X-API-Key"`
}

// GetPetParams supply the path parameter for /pets/{id}.
//
// swagger:parameters getPet
type GetPetParams struct {
	// in: path
	// required: true
	ID string `json:"id"`
}

// pathItemRefs is the standalone anchor for the path-item reference
// marker: it adds the shared X-Request-ID parameter to the /pets
// path-item as a $ref. (Not a struct → reference mode.)
//
// swagger:parameters /pets X-Request-ID
func pathItemRefs() {}

// ListPets lists pets, under the exact path /pets.
//
// swagger:route GET /pets pets listPets
// Responses:
//
//	200: description: OK
func ListPets() {}

// CreatePet creates a pet, under the exact path /pets. It declares no
// parameters of its own, so it inherits the /pets path-item params
// (X-API-Key required + the X-Request-ID $ref) unchanged.
//
// swagger:route POST /pets pets createPet
// Responses:
//
//	201: description: Created
func CreatePet() {}

// GetPet reads one pet, under the EXACT path /pets/{id}. It must NOT
// inherit the /pets path-item parameters (OAS2 has no path hierarchy).
//
// swagger:route GET /pets/{id} pets getPet
// Responses:
//
//	200: description: OK
func GetPet() {}
