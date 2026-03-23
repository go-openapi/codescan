// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_arrays exercises strfmt handling on named array and slice
// types, including the byte/bsonobjectid fast paths.
package strfmt_arrays

// Hash is a 32-byte array tagged as the byte swagger strfmt.
//
// swagger:strfmt byte
type Hash [32]byte

// ObjectID is a 12-byte array tagged as a BSON object id.
//
// swagger:strfmt bsonobjectid
type ObjectID [12]byte

// Signature is a named array that carries a generic strfmt tag.
//
// swagger:strfmt password
type Signature [64]byte

// Blob is a named byte slice tagged as the byte swagger strfmt.
//
// swagger:strfmt byte
type Blob []byte

// Token is a named slice tagged with a generic strfmt.
//
// swagger:strfmt uuid
type Token []string

// Carrier embeds all of the named array and slice types above.
//
// swagger:model
type Carrier struct {
	// required: true
	Hash Hash `json:"hash"`

	ObjectID ObjectID `json:"objectId"`

	Signature Signature `json:"signature"`

	Blob Blob `json:"blob"`

	Token Token `json:"token"`
}
