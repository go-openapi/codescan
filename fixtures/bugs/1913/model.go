// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug1913

// TeslaCar is a discriminated (polymorphic) type (go-swagger#1913).
//
// swagger:model
type TeslaCar interface {
	// The model of tesla car
	//
	// discriminator: true
	// swagger:name model
	Model() string

	// AutoPilot returns true when it supports autopilot
	// swagger:name autoPilot
	AutoPilot() bool
}

// ModelS version of the tesla car.
//
// swagger:model modelS
type ModelS struct {
	// swagger:allOf
	TeslaCar
	// The edition of this Model S
	Edition string `json:"edition"`
}

// ModelX version of the tesla car.
//
// swagger:model modelX
type ModelX struct {
	// swagger:allOf
	TeslaCar
	// Number of doors
	Doors int `json:"doors"`
}
