// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmtenum covers an enum whose declared type is written OVER another named type rather
// than directly over a basic one.
//
// `type Kind strfmt.UUID` reaches go/types with `string` as its underlying: the intermediate — and
// with it the `uuid` format the strfmt declaration carries — is not in that view. The same type
// without the enum annotation keeps its format, because the ordinary path resolves the declaration's
// right-hand side; the enum arm has to resolve it too, or the annotation silently costs the author
// their format.
package strfmtenum

import "github.com/go-openapi/strfmt"

// Kind is an enum over a strfmt type.
//
// swagger:enum Kind
type Kind strfmt.UUID

const (
	// KindPrimary is the primary kind.
	KindPrimary Kind = "0a8bcf1e-0000-0000-0000-000000000000"

	// KindSecondary is the secondary kind.
	KindSecondary Kind = "0a8bcf1e-1111-1111-1111-111111111111"
)

// Contact is an enum two redefinitions away from the strfmt: the walk to the right has to repeat.
//
// swagger:enum Contact
type Contact Address

// Address is the intermediate redefinition, carrying no annotation of its own.
type Address strfmt.Email

const (
	// ContactSupport is the support address.
	ContactSupport Contact = "support@example.com"
)

// Plain is an enum straight over a basic type: the control, whose format stays the basic one.
//
// swagger:enum Plain
type Plain string

const (
	// PlainOn is on.
	PlainOn Plain = "on"
)

// Labels carries the enums as schema properties.
//
// swagger:model Labels
type Labels struct {
	// The kind.
	Kind Kind `json:"kind"`

	// The contact.
	Contact Contact `json:"contact"`

	// The plain label.
	Plain Plain `json:"plain"`
}
