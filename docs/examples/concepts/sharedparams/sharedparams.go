// SPDX-License-Identifier: Apache-2.0

// Package sharedparams is the test-backed witness for the "Sharing
// parameters & responses" tutorial (go-swagger#2632). It declares a small
// pet API that reuses a header parameter and an error response across
// several operations through the spec-level shared namespace
// (#/parameters and #/responses).
package sharedparams

// snippet:shared

// CommonHeaders registers a reusable header parameter at the spec top
// level, #/parameters/X-Request-ID. The bare `*` target is register-only:
// it publishes the parameter but does not, by itself, attach it to any
// operation.
//
// swagger:parameters *
type CommonHeaders struct {
	// RequestID correlates a request across services.
	//
	// in: header
	RequestID string `json:"X-Request-ID"` //nolint:tagliatelle // canonical HTTP header name
}

// AuthHeader registers #/parameters/X-API-Key and, in the same breath,
// $ref's it into the createPet operation — the convenient `* <opid>`
// form for a small spec.
//
// swagger:parameters * createPet
type AuthHeader struct {
	// APIKey authorises access.
	//
	// in: header
	// required: true
	APIKey string `json:"X-API-Key"` //nolint:tagliatelle // canonical HTTP header name
}

// endsnippet:shared

// snippet:sharedresponse

// ErrorResponse is the common error envelope returned by every operation.
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

// endsnippet:sharedresponse

// CreatePetParams carries the request body for createPet; it is inlined
// into the operation the normal way and sits alongside the $ref'd
// X-API-Key header.
//
// swagger:parameters createPet
type CreatePetParams struct {
	// in: body
	// required: true
	Body Pet `json:"body"`
}

// Pet is the body model.
//
// swagger:model
type Pet struct {
	// Name of the pet.
	Name string `json:"name"`
}

// snippet:routes

// ListPets lists pets.
//
// The standalone reference marker on the next line pulls the shared
// X-Request-ID parameter into this operation as a $ref. This is the
// scaling channel: the shared struct need not enumerate every operation
// that wants the parameter.
//
// swagger:route GET /pets pets listPets
// swagger:parameters listPets X-Request-ID
// Responses:
//
//	default: ErrorResponse
func ListPets() {}

// CreatePet creates a pet. Its X-API-Key comes from AuthHeader
// (#/parameters/X-API-Key, $ref'd via `* createPet`); its body comes from
// the inlined CreatePetParams.
//
// swagger:route POST /pets pets createPet
// Responses:
//
//	default: ErrorResponse
func CreatePet() {}

// endsnippet:routes

// snippet:pathitem

// TenantHeader inlines a required header into the /pets/{id} path-item
// itself, so every operation under that exact path inherits it. The
// target is a literal path, and matching is exact — OAS2 has no path
// hierarchy, so this does NOT apply to /pets.
//
// swagger:parameters /pets/{id}
type TenantHeader struct {
	// Tenant scopes the request to a customer.
	//
	// in: header
	// required: true
	Tenant string `json:"X-Tenant"` //nolint:tagliatelle // canonical HTTP header name
}

// GetPet fetches one pet. It declares no header of its own; X-Tenant
// reaches it through the /pets/{id} path-item parameter above.
//
// swagger:route GET /pets/{id} pets getPet
// Responses:
//
//	default: ErrorResponse
func GetPet() {}

// endsnippet:pathitem
