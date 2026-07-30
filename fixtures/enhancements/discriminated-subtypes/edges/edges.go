// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package edges holds the embed shapes the reverse index must classify, in a
// family NO route references — so it also locks the other half of the gate: a
// discriminated base that never enters the reachable set pulls nothing in.
package edges

// Vehicle is a struct base.
//
// Its discriminator sits on a property rather than on an interface member.
//
// swagger:model Vehicle
type Vehicle struct {
	// The kind of vehicle
	//
	// discriminator: true
	Kind string `json:"kind"`
}

// VehicleAlias names the base through an alias, which the index resolves to the
// same base.
type VehicleAlias = Vehicle

// Bike declares the base as an allOf member BY POINTER.
//
// The index unwraps the pointer.
//
// swagger:model Bike
type Bike struct {
	// swagger:allOf
	*Vehicle

	// Number of gears
	Gears int `json:"gears"`
}

// Trike reaches the base through an alias.
//
// swagger:model Trike
type Trike struct {
	// swagger:allOf
	VehicleAlias

	// Number of wheels
	Wheels int `json:"wheels"`
}

// Hidden declares the base as an allOf member, but the embed is dropped.
//
// The embed carries an ignore annotation, so no relation exists to index.
//
// swagger:model Hidden
type Hidden struct {
	// swagger:ignore
	// swagger:allOf
	*Vehicle

	// Whatever
	Whatever string `json:"whatever"`
}

// Plain embeds the base with no annotation at all.
//
// Its properties are inlined and no allOf member is emitted, so it is not a
// subtype.
//
// swagger:model Plain
type Plain struct {
	Vehicle

	// Extra field
	Extra string `json:"extra"`
}
