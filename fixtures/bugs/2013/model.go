// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2013

// Base is embedded; the #2013 panic (index out of range in SetEnum) occurred in
// buildEmbedded while parsing an enum on a promoted field.
type Base struct {
	// Kind of thing.
	//
	// enum: a,b,c
	Kind string `json:"kind"`
}

// Thing embeds Base and carries its own enum field.
//
// swagger:model
type Thing struct {
	Base
	// Status of the thing.
	//
	// enum: open,closed
	Status string `json:"status"`
}
