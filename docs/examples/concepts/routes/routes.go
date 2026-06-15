// SPDX-License-Identifier: Apache-2.0

// Package routes holds the annotated declarations used by the "Routes &
// operations" tutorial. routes_test.go scans this package and writes the
// per-annotation golden fragments (path items, parameters, responses) the
// tutorial renders, so the documentation can never drift from real output.
package routes

import "io"

// Pet is the model the operations below produce and consume.
//
// swagger:model
type Pet struct {
	// ID is the unique identifier.
	ID int64 `json:"id"`

	// Name is the pet's display name.
	Name string `json:"name"`
}

// snippet:route

// swagger:route GET /pets pets listPets
//
// Lists pets in the store, optionally filtered by tag.
//
// responses:
//
//	200: petsResponse
//	default: errorResponse

// endsnippet:route

// snippet:operation

// swagger:operation GET /pets/{id} pets getPet
//
// ---
// summary: Get a pet by ID.
// parameters:
//   - name: id
//     in: path
//     required: true
//     type: integer
//     format: int64
// responses:
//   '200':
//     description: the requested pet
//     schema:
//       $ref: '#/definitions/Pet'
//   default:
//     $ref: '#/responses/errorResponse'

// endsnippet:operation

// snippet:parameters

// ListPetsParams is the parameter set for the listPets operation. Each field
// becomes one parameter; the operation IDs after swagger:parameters name the
// operations the set applies to.
//
// swagger:parameters listPets
type ListPetsParams struct {
	// Tag filters pets by tag.
	//
	// in: query
	Tag string `json:"tag"`

	// Limit caps the number of results.
	//
	// in: query
	// minimum: 1
	// maximum: 100
	Limit int32 `json:"limit"`
}

// endsnippet:parameters

// snippet:response

// PetsResponse is the list returned by listPets.
//
// swagger:response petsResponse
type PetsResponse struct {
	// in: body
	Body []Pet
}

// ErrorResponse is the default error payload.
//
// swagger:response errorResponse
type ErrorResponse struct {
	// in: body
	Body struct {
		// Message is a human-readable error message.
		Message string `json:"message"`
	}
}

// endsnippet:response

// snippet:file

// swagger:route POST /pets/{id}/photo pets uploadPetPhoto
//
// responses:
//
//	200: petsResponse

// UploadParams is the multipart upload for the uploadPetPhoto operation.
//
// swagger:parameters uploadPetPhoto
type UploadParams struct {
	// Photo is the image to upload.
	//
	// in: formData
	// swagger:file
	Photo io.ReadCloser `json:"photo"`
}

// endsnippet:file

// snippet:externaldocs

// swagger:route GET /pets/search pets searchPets
//
// Searches pets. The operation links out to external documentation.
//
// externalDocs:
//   description: Search guide
//   url: https://example.com/docs/search
//
// responses:
//
//	200: petsResponse

// CatalogEntry carries externalDocs at the schema level (the link rides the
// definition) and on its fields. (On a simple-schema parameter externalDocs is
// dropped with a diagnostic: it is a full-Schema-only keyword.)
//
// externalDocs: {description: "Catalog schema reference", url: "https://example.com/docs/catalog"}
//
// swagger:model
type CatalogEntry struct {
	// SKU is the catalog identifier.
	SKU string `json:"sku"`

	// Vendor is a plain field: externalDocs attaches directly to the property.
	//
	// externalDocs: {description: "Vendor field docs", url: "https://example.com/docs/vendor"}
	Vendor string `json:"vendor"`

	// Supplier is a $ref'd field: its sibling externalDocs lifts onto the
	// field's allOf compound (a bare $ref cannot carry sibling keywords).
	//
	// externalDocs: {description: "Supplier docs", url: "https://example.com/docs/supplier"}
	Supplier Supplier `json:"supplier"`
}

// Supplier is referenced by CatalogEntry.Supplier.
//
// swagger:model
type Supplier struct {
	// Name is the supplier name.
	Name string `json:"name"`
}

// endsnippet:externaldocs
