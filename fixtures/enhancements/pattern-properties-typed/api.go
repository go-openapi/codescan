// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package patternpropstyped exercises the typed patternProperties marker:
// quoted-regex to typed-value pairs (a primitive or a model $ref), RE2-hygiene
// checking, and the lowest-priority precedence rule.
package patternpropstyped

// Item is referenced as a patternProperties value schema.
//
// swagger:model Item
type Item struct {
	Name string `json:"name"`
}

// TypedPatterns maps property-name patterns to typed value schemas, alongside a
// named property.
//
// swagger:model TypedPatterns
// swagger:patternProperties "^x-": string, "^\d+$": integer, "^item-": Item
type TypedPatterns struct {
	Known string `json:"known"`
}

// InvalidTyped carries a regex that does not compile under RE2; the value is
// preserved but a diagnostic is raised.
//
// swagger:model InvalidTyped
// swagger:patternProperties "[unclosed(": string
type InvalidTyped struct {
	A string `json:"a"`
}

// Contradiction resolves to a string via swagger:type; patternProperties is the
// lowest-priority annotation, so it is dropped with a diagnostic.
//
// swagger:model Contradiction
// swagger:type string
// swagger:patternProperties "^x-": string
type Contradiction struct {
	A string `json:"a"`
}
