// SPDX-License-Identifier: Apache-2.0

// Package polymorphism holds the annotated declarations used by the "Polymorphic
// models" tutorial. polymorphism_test.go scans it and writes the per-type golden
// fragments the tutorial renders, so the documentation can never drift from the
// scanner's real output.
package polymorphism

// snippet:base

// Pet is the polymorphic base type. The field marked `discriminator: true` names
// the property whose value tells a consumer which concrete subtype a payload is:
// codescan writes that property's name onto the schema's `discriminator`, and a
// discriminator property must be `required`.
//
// swagger:model
type Pet struct {
	// PetType selects the concrete subtype — its value is the subtype's
	// definition name (e.g. "Cat" or "Dog").
	//
	// discriminator: true
	// required: true
	PetType string `json:"petType"`

	// Name is common to every pet.
	//
	// required: true
	Name string `json:"name"`
}

// endsnippet:base

// snippet:children

// Cat is a Pet subtype. `swagger:allOf` composes it as
// `allOf: [ $ref Pet, {its own fields} ]` — the composition Swagger 2.0
// polymorphism builds on.
//
// swagger:model
type Cat struct {
	// swagger:allOf
	Pet

	// HuntingSkill is how the cat hunts.
	HuntingSkill string `json:"huntingSkill"`
}

// Dog is a second Pet subtype.
//
// swagger:model
type Dog struct {
	// swagger:allOf
	Pet

	// PackSize is the size of the dog's pack.
	PackSize int32 `json:"packSize"`
}

// endsnippet:children
