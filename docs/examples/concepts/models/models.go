// SPDX-License-Identifier: Apache-2.0

// Package models holds the annotated types used by the "Model definitions"
// tutorial. Each region demonstrates one model annotation; models_test.go scans
// this package and writes the per-type golden fragments the tutorial renders, so
// the documentation can never drift from the scanner's real output.
package models

// snippet:model

// Pet is a single pet in the store.
//
// swagger:model
type Pet struct {
	// ID is the unique identifier.
	ID int64 `json:"id"`

	// Name is the pet's display name.
	Name string `json:"name"`

	// Tags categorise the pet.
	Tags []string `json:"tags,omitempty"`
}

// endsnippet:model

// snippet:strfmt

// MAC is a hardware address rendered as a colon-separated hex string.
//
// swagger:strfmt mac
type MAC string

func (m MAC) MarshalText() ([]byte, error)  { return []byte(m), nil }
func (m *MAC) UnmarshalText(b []byte) error { *m = MAC(b); return nil }

// Device exposes a strfmt-typed field: wherever MAC appears it renders inline
// as {type: string, format: mac}.
//
// swagger:model
type Device struct {
	// Addr is the hardware address.
	Addr MAC `json:"addr"`
}

// endsnippet:strfmt

// snippet:enum

// Priority is the urgency level on a task.
//
// swagger:enum Priority
type Priority string

const (
	// PriorityLow is for tasks that can wait.
	PriorityLow Priority = "low"
	// PriorityMedium is the default.
	PriorityMedium Priority = "medium"
	// PriorityHigh is for tasks that must run soon.
	PriorityHigh Priority = "high"
)

// Task is a unit of work carrying an enum-typed field. Referencing Priority
// from a model is what makes the enum reachable, and so emitted.
//
// swagger:model
type Task struct {
	// Priority is the task's urgency.
	Priority Priority `json:"priority"`
}

// endsnippet:enum

// snippet:allof

// Animal is one abstract base.
//
// swagger:model
type Animal struct {
	// Kind discriminates the animal.
	Kind string `json:"kind"`
}

// Tagged is a second reusable base.
//
// swagger:model
type Tagged struct {
	// Tags label the resource.
	Tags []string `json:"tags"`
}

// Dog composes two base models plus its own fields: each embedded base becomes
// a $ref arm of the allOf, and the struct's own (non-embedded) fields — which
// are new and cannot be a $ref — form the final inline arm.
//
// swagger:model
type Dog struct {
	// swagger:allOf
	Animal

	// swagger:allOf
	Tagged

	// Breed is the dog's breed.
	Breed string `json:"breed"`
}

// endsnippet:allof

// snippet:type

// ULID is a 128-bit identifier stored as bytes but rendered as a string.
//
// swagger:type string
type ULID [16]byte

// Token carries a field whose inferred type is overridden, inline.
//
// swagger:model
type Token struct {
	// ID renders as a string despite its [16]byte Go type.
	ID ULID `json:"id"`
}

// endsnippet:type

// snippet:typefield

// RawID is a custom 16-byte identifier — an array under the hood, so left to
// itself a field of this type would render as an array of integers.
type RawID [16]byte

// Coupon overrides the type of a single field directly on the field doc — no
// wrapper-type annotation. Code publishes as a bare string while RawID is left
// untouched everywhere else it appears.
//
// swagger:model
type Coupon struct {
	// Code is an opaque identifier published as a string.
	//
	// swagger:type string
	Code RawID `json:"code"`

	// Amount is the discount in cents.
	Amount int64 `json:"amount"`
}

// endsnippet:typefield

// snippet:name

// Car is exposed as a schema via its method set. Interface methods cannot carry
// a json tag, so by default each property takes the camelCased method name;
// swagger:name overrides that where the default is not what you want.
//
// swagger:model
type Car interface {
	// Maker is the manufacturer. With no override the property is the
	// camelCased method name, "maker".
	Maker() string

	// StructType is the polymorphic class. Without the override the property
	// would be "structType"; swagger:name publishes it as "jsonClass".
	//
	// swagger:name jsonClass
	StructType() string
}

// endsnippet:name

// snippet:namekeyword

// Account shows the universal name: keyword renaming model struct fields. The
// same keyword used on parameters and response headers also sets a property key
// here, winning over a json tag, the legacy swagger:name annotation, and the Go
// field name.
//
// swagger:model
type Account struct {
	// Bal has no json tag; the keyword sets the property key directly.
	//
	// name: balance
	Bal float64

	// Currency carries both naming forms; the keyword wins over the
	// legacy annotation and the json tag.
	//
	// name: currencyCode
	// swagger:name legacyCurrency
	Currency string `json:"currency"`
}

// endsnippet:namekeyword

// snippet:ignore

// Secret never reaches the spec.
//
// swagger:ignore
type Secret struct {
	// Token is internal.
	Token string `json:"token"`
}

// endsnippet:ignore
