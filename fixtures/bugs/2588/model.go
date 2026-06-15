// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package bug2588

// A is an interface.
type A interface {
	Do()
}

// B is a named type whose underlying type is an interface (#2588).
type B A

// C embeds B (an interface-underlying named type).
//
// swagger:model
type C struct {
	B
}
