// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_arrays exercises strfmt handling on named array and slice
// types. Whether a format describes the whole value or its items is decided by
// the ELEMENT type: byte and rune sequences are string-like and take the format
// on the schema, everything else takes it on the items.
package strfmt_arrays

// Hash is a 32-byte array tagged as the byte swagger strfmt.
//
// swagger:strfmt byte
type Hash [32]byte

// ObjectID is a 12-byte array tagged as a BSON object id.
//
// swagger:strfmt bsonobjectid
type ObjectID [12]byte

// Signature is a named byte array carrying a format that is not one of the two
// names the old allowlist knew. It is still a byte sequence, so the format
// describes the whole value — it used to emit an array of 64 password strings.
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
