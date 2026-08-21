// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build go1.27

package uuidmodels

import "uuid"

// Order carries the stdlib uuid.UUID in every position the schema builder can reach it from.
//
// swagger:model Order
type Order struct {
	// ID is a plain field.
	ID uuid.UUID `json:"id"`

	// PtrID is a pointer, peeled before recognition.
	PtrID *uuid.UUID `json:"ptrId"`

	// IDs exercises the items chain.
	IDs []uuid.UUID `json:"ids"`

	// ByName exercises a map value; the key is a plain string.
	ByName map[string]uuid.UUID `json:"byName"`

	// Keyed exercises uuid.UUID as a map KEY: it marshals to a JSON object because the key is a
	// TextMarshaler, and the key type itself is not rendered in Swagger 2.0.
	Keyed map[uuid.UUID]string `json:"keyed"`

	// Named refs the named type declared below.
	Named OrderID `json:"named"`

	// Overridden proves the explicit classifier still beats the recognizer.
	//
	// swagger:strfmt date
	Overridden uuid.UUID `json:"overridden"`
}

// OrderID is a named type whose underlying type is the stdlib uuid.UUID.
//
// swagger:model OrderID
type OrderID uuid.UUID

// AliasID is an ALIAS of the stdlib uuid.UUID: it dissolves at use sites.
type AliasID = uuid.UUID

// Tagged uses the alias.
//
// swagger:model Tagged
type Tagged struct {
	// Alias is declared through the alias above.
	Alias AliasID `json:"alias"`
}

// Embedder embeds the stdlib uuid.UUID.
//
// Witness for the embedded arm, which the fuzzy name heuristic never reached (it is caller-gated to
// buildFromTextMarshal). uuid.UUID is an array underneath, so the embed promotes nothing and Go
// keeps it as an ordinary key named after the type: the schema carries a UUID property, built
// through the identity recognizer like any other member.
//
// The promoted MarshalText makes the DEFAULT marshaller render the whole struct as a bare string,
// dropping Name from the wire. codescan does not model that — an embed means composition here — see
// the schema builder README, section embed-marshaller.
//
// swagger:model Embedder
type Embedder struct {
	uuid.UUID

	// Name is NOT on the wire, whatever the emitted schema says.
	Name string `json:"name"`
}
