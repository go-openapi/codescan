// SPDX-License-Identifier: Apache-2.0

// Package inlineresponses holds the annotated declarations for the "Inline
// response bodies" how-to. inlineresponses_test.go scans it and writes the
// golden path item the guide renders, so the documentation can never drift from
// the scanner's real output.
package inlineresponses

// Pet is the model the inline responses reference.
//
// swagger:model
type Pet struct {
	// ID is the unique identifier.
	ID int64 `json:"id"`

	// Name is the pet's display name.
	Name string `json:"name"`
}

// snippet:inline

// swagger:route GET /pets pets listPets
//
// Lists pets. Each response is declared inline with the body: sub-language — a
// primitive, an array of a model, or a single model $ref — so no wrapper
// response type is needed. Trailing words become the response description.
//
//	Responses:
//	  200: body:[]Pet the list of pets
//	  400: body:string an error message
//	  default: body:Pet a single pet

// endsnippet:inline
