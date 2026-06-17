// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug3214 reproduces go-swagger issue #3214: a struct field that
// references a named primitive type whose doc comment carries both prose
// and an inline `enum:` annotation.
//
// The historical bug: the referenced typed primitive was only parsed for
// title/description and the `enum:` declaration line was wrongly swallowed
// into the description rather than parsed into enum values. The expected
// behaviour — locked by the golden — is that the enum values are parsed
// and the prose is preserved untouched (no `enum:` text leaking into
// title or description).
package bug3214

// Order references typed primitives.
//
// swagger:model Order
type Order struct {
	// The current state of the order.
	State State `json:"state"`

	// The currency the order is billed in.
	Currency Currency `json:"currency"`
}

// State represents the state of an order.
// enum: ["created","processed"]
type State string

// Currency is the billing currency of an order.
//
// It is rendered as an ISO-4217 alphabetic code. This second prose
// paragraph proves that a full description paragraph survives intact and
// that the annotation declaration line below is not folded into it.
//
// enum: ["USD","EUR","JPY"]
type Currency string
