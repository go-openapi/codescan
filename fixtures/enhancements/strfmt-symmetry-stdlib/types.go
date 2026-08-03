// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package strfmt_symmetry_stdlib exercises the strfmt annotation over a stdlib
// type that the builder ALSO recognizes on its own — the precedence cell.
//
// This is not simply a missing classifier call like the rest of the matrix. On
// the named half the classifier wins, because `applyStdlibSpecials` is keyed on
// the declaration's own identity (`StampNamed` is not `time.Time`) and so does
// not fire. On the alias half the dissolve reaches the stdlib type itself, the
// recognizer fires, and the author's annotation is overruled by a DIFFERENT
// answer rather than by no answer at all.
//
// The contract is that the author always wins: the annotation is the escape hatch
// for formats the library cannot infer, so a recognizer is a default for
// un-annotated code and never an override. Kept in its own package because these
// goldens move with that precedence rule rather than with the alias dispatch.
//
// See [§special-types](../../../internal/builders/schema/README.md#special-types).
package strfmt_symmetry_stdlib

import (
	"encoding/json"
	"time"
)

// StampNamed is a named type over time.Time, annotated as a plain date.
//
// swagger:strfmt date
type StampNamed time.Time

// StampAlias is an alias over time.Time, same annotation.
//
// swagger:strfmt date
type StampAlias = time.Time

// RawNamed is a named type over json.RawMessage, annotated as base64 bytes.
//
// swagger:strfmt byte
type RawNamed json.RawMessage

// RawAlias is an alias over json.RawMessage, same annotation.
//
// swagger:strfmt byte
type RawAlias = json.RawMessage

// Envelope reaches both pairs from a field site.
//
// swagger:model Envelope
type Envelope struct {
	// FieldTimeNamed is the time pair's named half.
	FieldTimeNamed StampNamed `json:"fieldTimeNamed"`

	// FieldTimeAlias is the time pair's alias half.
	FieldTimeAlias StampAlias `json:"fieldTimeAlias"`

	// FieldRawNamed is the raw-message pair's named half.
	FieldRawNamed RawNamed `json:"fieldRawNamed"`

	// FieldRawAlias is the raw-message pair's alias half.
	FieldRawAlias RawAlias `json:"fieldRawAlias"`
}
