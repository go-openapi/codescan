// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1737

// ObjectID stands in for an external named type (e.g. bson.ObjectId) that
// codescan turns into its own definition (go-swagger#1737).
//
// swagger:model
type ObjectID struct {
	Hex string `json:"hex"`
}

// AResponseBody carries a field whose type resolves to a $ref. The reporter
// found the field-level description was dropped because a Swagger 2.0 property
// that is a bare $ref cannot carry sibling keys.
type AResponseBody struct {
	// SomeID description that the reporter wants to keep.
	SomeID ObjectID `json:"someId"`
}

// AResponse is the body response wrapping AResponseBody.
//
// swagger:response aResponse
type AResponse struct {
	// in: body
	Body AResponseBody
}
