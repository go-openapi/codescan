// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_symmetry_simpleschema exercises the strfmt annotation at the
// SimpleSchema dispatch sites — non-body parameters and response headers, where
// OAS v2 forbids `$ref` so the builder runs with `simpleSchema` set and the
// `refModel` gate flips (`schema.go:475`).
//
// Body parameters and response bodies are deliberately absent: both are
// full-schema locations that re-enter `buildFromType` exactly as a struct field
// does, so `strfmt-symmetry-core` already covers that dispatch.
//
// Each pair is the same right-hand side with the same annotation, differing only
// by the `=`, reached from the same parameter set / response.
//
// See [§aliases](../../../internal/builders/schema/README.md#aliases).
package strfmt_symmetry_simpleschema

// FmtBasicNamed is a named type over a primitive, carrying a format.
//
// swagger:strfmt isbn
type FmtBasicNamed string

// FmtBasicAlias is an alias over the same primitive, same format.
//
// swagger:strfmt isbn
type FmtBasicAlias = string

// FmtSliceNamed is a named type over a byte slice, carrying the whole-schema
// "byte" format special.
//
// swagger:strfmt byte
type FmtSliceNamed []byte

// FmtSliceAlias is an alias over the same byte slice, same format.
//
// swagger:strfmt byte
type FmtSliceAlias = []byte

// SimpleParams carries the non-body parameter cells.
//
// swagger:parameters simpleOp
type SimpleParams struct {
	// QueryBasicNamed is the basic pair's named half in query position.
	//
	// in: query
	QueryBasicNamed FmtBasicNamed `json:"queryBasicNamed"`

	// QueryBasicAlias is the basic pair's alias half in query position.
	//
	// in: query
	QueryBasicAlias FmtBasicAlias `json:"queryBasicAlias"`

	// QuerySliceNamed is the slice pair's named half in query position.
	//
	// in: query
	QuerySliceNamed FmtSliceNamed `json:"querySliceNamed"`

	// QuerySliceAlias is the slice pair's alias half in query position.
	//
	// in: query
	QuerySliceAlias FmtSliceAlias `json:"querySliceAlias"`
}

// SimpleResponse carries the response-header cells. Non-body fields of a
// response struct become headers, which are SimpleSchema locations too.
//
// swagger:response simpleResponse
type SimpleResponse struct {
	// HeaderBasicNamed is the basic pair's named half in header position.
	//
	// in: header
	HeaderBasicNamed FmtBasicNamed `json:"headerBasicNamed"`

	// HeaderBasicAlias is the basic pair's alias half in header position.
	//
	// in: header
	HeaderBasicAlias FmtBasicAlias `json:"headerBasicAlias"`

	// HeaderSliceNamed is the slice pair's named half in header position.
	//
	// in: header
	HeaderSliceNamed FmtSliceNamed `json:"headerSliceNamed"`

	// HeaderSliceAlias is the slice pair's alias half in header position.
	//
	// in: header
	HeaderSliceAlias FmtSliceAlias `json:"headerSliceAlias"`
}

// SimpleHandler binds the parameter set and the response to a real operation, so
// `paths` populates and the SimpleSchema locations are observable.
//
// swagger:route GET /simple symmetry simpleOp
//
// Responses:
//
//	200: simpleResponse
func SimpleHandler() {}
