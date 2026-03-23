// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package interface_methods exercises the processInterfaceMethod and
// processAnonInterfaceMethod branches of the schema builder: non-exported
// methods, methods with parameters, methods with multiple returns,
// swagger:ignore methods, swagger:name overrides, swagger:strfmt returns,
// and pointer returns that may be x-nullable.
package interface_methods

import "time"

// Audited is a small named interface that is embedded with swagger:allOf
// into richer interfaces below.
//
// swagger:model
type Audited interface {
	// CreatedAt is the creation timestamp.
	CreatedAt() time.Time

	// UpdatedAt is the update timestamp.
	UpdatedAt() time.Time
}

// UserProfile is a read-only view over a user, exposed as a schema via its
// method set. Each method exercises a distinct branch of
// processInterfaceMethod.
//
// swagger:model
type UserProfile interface {
	// swagger:allOf
	Audited

	// ID is the user identifier.
	//
	// required: true
	// min: 1
	ID() int64

	// Name is re-exposed as "fullName" in JSON.
	//
	// swagger:name fullName
	Name() string

	// Email is formatted as an email strfmt.
	//
	// swagger:strfmt email
	Email() string

	// Bio is a nullable pointer string.
	Bio() *string

	// Tags returns the user's labels.
	Tags() []string

	// Profile returns nested structured data.
	Profile() map[string]string

	// swagger:ignore
	//
	// Secret is deliberately omitted from the schema.
	Secret() string

	// WithParams takes an argument and is therefore not a valid property
	// accessor; the scanner must skip it.
	WithParams(x int) string

	// WithMultiReturn returns multiple values and is also skipped.
	WithMultiReturn() (string, error)

	// WithNoReturn returns nothing and is skipped.
	WithNoReturn()

	// notExported is an unexported method that must be skipped.
	notExported() int
}

// Public exposes just a single scalar so we get a minimal, deterministic
// companion to assert the default code path.
//
// swagger:model
type Public interface {
	// Kind names the public flavor.
	Kind() string
}

// WithAnonEmbed embeds an anonymous inline interface so that the scanner
// walks processAnonInterfaceMethod for its methods. This exercises the
// buildAnonymousInterface call site inside processEmbeddedType.
//
// swagger:model
type WithAnonEmbed interface {
	// swagger:allOf
	interface {
		// AuditTrail is exposed via the anonymous embedded interface.
		//
		// swagger:name audit
		AuditTrail() string

		// ExternalID is tagged as uuid so the anon-method strfmt branch
		// is exercised.
		//
		// swagger:strfmt uuid
		ExternalID() string

		// Revision is a nullable pointer return for x-nullable coverage.
		Revision() *int

		// swagger:ignore
		IgnoredByAnon() string

		internalOnly() bool

		WithArgs(y int) string

		WithMulti() (string, error)
	}

	// Kind names the root flavor.
	Kind() string
}
