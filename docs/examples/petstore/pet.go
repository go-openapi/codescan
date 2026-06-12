// SPDX-License-Identifier: Apache-2.0

package petstore

// snippet:route

// swagger:route GET /pets pets listPets
//
// Lists all the pets in the store.
//
// responses:
//
//	200: petsResponse
// endsnippet:route

// snippet:model

// Pet is a single pet in the store.
//
// swagger:model Pet
type Pet struct {
	// The id of the pet.
	//
	// required: true
	// minimum: 1
	ID int64 `json:"id"`

	// The name of the pet.
	//
	// required: true
	// min length: 1
	Name string `json:"name"`

	// The tags associated with this pet.
	Tags []string `json:"tags,omitempty"`
}

// endsnippet:model

// petsResponse is the list of pets returned by listPets.
//
// swagger:response petsResponse
//
//nolint:unused // referenced by the swagger:response annotation above, not from Go code
type petsResponse struct {
	// in: body
	Body []Pet
}
