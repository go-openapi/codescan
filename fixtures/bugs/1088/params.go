// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1088

// Label is a named primitive (underlying string).
type Label string

// Ele is an object element.
type Ele struct {
	SomeID   int    `json:"someId"`
	SomeName string `json:"someName"`
}

// Request declares array query parameters. An array IS valid in a query
// parameter, but the items form a Swagger 2.0 *simple schema* — it may not be a
// $ref. The scanner used to emit `items: {$ref}` (invalid; the editor rejects
// it). The $ref must dissolve: a named primitive expands to its type
// (Label -> string); an object element has no valid simple-schema form, so it
// dissolves to an empty items schema with a diagnostic.
//
// swagger:parameters arrayReq
type Request struct {
	// in: query
	Tags []Label `json:"tags"`

	// in: query
	Arr []Ele `json:"arr"`
}

// ArrayHeaders carries response headers. A response header is a Swagger 2.0
// simple schema too, so array-of-object headers are subject to the same
// constraint as query parameters (go-swagger#1088).
//
// swagger:response arrayResp
type ArrayHeaders struct {
	// in: header
	XTags []Label `json:"X-Tags"`

	// in: header
	XObjs []Ele `json:"X-Objs"`
}

// Things uses the arrayReq parameters and the arrayResp headers.
//
// swagger:route GET /things things arrayReq
//
// List things.
//
//	Responses:
//	  200: arrayResp
func Things() {}
