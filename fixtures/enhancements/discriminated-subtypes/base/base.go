// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package base holds the polymorphic bases, in a package of their own so that
// subtypes can live in more than one other package without an import cycle.
package base

// TeslaCar is the discriminated (polymorphic) base.
//
// swagger:model TeslaCar
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

// PlainBase is a base WITHOUT a discriminator.
//
// Its family is not polymorphic, so nothing is reverse-pulled for it.
//
// swagger:model PlainBase
type PlainBase interface {
	// swagger:name kind
	Kind() string
}
