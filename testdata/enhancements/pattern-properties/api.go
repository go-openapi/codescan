// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package patternproperties exercises the patternProperties object schema
// keyword: a string-typed annotation whose argument is a regex. Each line
// adds one regex → empty-schema entry to the schema's patternProperties
// map, and the regex is RE2-hygiene-checked (mirroring the pattern keyword).
package patternproperties

// ValidPatterns is a free-form object constrained by two property-name
// regexes.
//
// patternProperties: ^[a-z]+$
// patternProperties: ^x-
//
// swagger:model ValidPatterns
type ValidPatterns map[string]any

// InvalidPattern carries a regex that does not compile under Go's RE2
// engine. The value is preserved on the schema but a CodeInvalidAnnotation
// diagnostic is emitted (the keyword is NOT dropped silently).
//
// patternProperties: [unclosed(
//
// swagger:model InvalidPattern
type InvalidPattern map[string]any
