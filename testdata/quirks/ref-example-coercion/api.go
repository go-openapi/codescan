// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package refexamplecoercion exercises doc-quirk G3: a JSON-literal
// example: / default: on a field whose type is a $ref must be coerced
// into a structured value on the allOf override arm, exactly as it is on
// a plain (non-ref) field. Before the fix the literal rode the override
// arm as a raw string ("{\"name\":\"x\"}" rather than {"name":"x"}).
//
// The referenced model (Pet) gives the field a $ref, so its sibling
// keywords (example, default) land on allOf[1]; the coercion that runs on
// the direct-field path must run here too.
package refexamplecoercion

// Pet is a referenced model so that fields typed by it carry a $ref.
//
// swagger:model Pet
type Pet struct {
	// the pet's name
	Name string `json:"name"`
}

// Tag is a referenced model used for the array-literal example case.
//
// swagger:model Tag
type Tag struct {
	// the tag value
	Value string `json:"value"`
}

// Holder carries fields typed by other models, each annotated with a
// JSON-literal example/default that must coerce structurally.
//
// swagger:model Holder
type Holder struct {
	// a single pet, with a JSON-object example riding the override arm
	// example: {"name":"Rex"}
	Pet Pet `json:"pet"`

	// a single pet, with a JSON-object default riding the override arm
	// default: {"name":"Default"}
	DefaultPet Pet `json:"defaultPet"`

	// a tag, with a JSON-array example riding the override arm
	// example: [{"value":"a"},{"value":"b"}]
	Tags Tag `json:"tags"`

	// a scalar example on a $ref'd field is left as a plain string (the
	// referenced type is unknown on the override arm; only JSON object /
	// array literals are coerced).
	// example: plain-scalar
	Plain Pet `json:"plain"`
}

// swagger:route GET /holder things getHolder
//
// Get a holder.
//
// responses:
//
//	200: holderResponse
func getHolder() {}

// holderResponse wraps the Holder model.
//
// swagger:response holderResponse
type holderResponse struct {
	// in: body
	Body Holder
}
