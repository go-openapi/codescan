// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_calibration_stdlib exercises the schema builder's
// alias handling for stdlib-special RHS types (`time.Time`,
// `error`, `json.RawMessage`, `any`). `applyStdlibSpecials`
// recognises these types at the schema layer; this fixture
// asserts the recognizer fires across the `buildDeclAlias`
// dispatch paths regardless of mode.
//
// Cells exercised:
//
//   - decl × annotated × stdlib(time.Time) → Timestamp
//   - decl × annotated × stdlib(error) → Err
//   - decl × annotated × stdlib(json.RawMessage) → Raw
//   - decl × unannotated × stdlib(time.Time) → SilentTime
//   - field × annotated alias → Envelope.at / .failure / .payload
//   - field × unannotated alias → Envelope.silent
//
// See the schema builder's alias-handling and special-types
// contracts:
// [§aliases](../../../internal/builders/schema/README.md#aliases),
// [§special-types](../../../internal/builders/schema/README.md#special-types).
package alias_calibration_stdlib

import (
	"encoding/json"
	"time"
)

// Timestamp is an alias to time.Time exposed as a top-level model.
// Stdlib-special: `applyStdlibSpecials` recognises it as
// `{type:string, format:date-time}`.
//
// swagger:model Timestamp
type Timestamp = time.Time

// Err is an alias to the predeclared error interface exposed as a
// top-level model. Stdlib-special.
//
// swagger:model Err
type Err = error

// Raw is an alias to json.RawMessage exposed as a top-level model.
// Stdlib-special.
//
// swagger:model Raw
type Raw = json.RawMessage

// SilentTime is an alias to time.Time with NO swagger:model
// annotation — reachable only via Envelope.Silent. Tests that the
// annotation gate at use sites holds for stdlib-special RHS just
// as it does for primitive RHS: the unannotated alias dissolves
// and the recognizer's canonical shape lands inline at the
// `Envelope.silent` field site.
type SilentTime = time.Time

// Envelope uses all four aliases as field types: three annotated,
// one unannotated. Exercises the field-reach context for each.
//
// swagger:model Envelope
type Envelope struct {
	// At — annotated stdlib-time alias in field position.
	At Timestamp `json:"at"`

	// Failure — annotated stdlib-error alias in field position.
	Failure Err `json:"failure,omitempty"`

	// Payload — annotated json.RawMessage alias in field position.
	Payload Raw `json:"payload"`

	// Silent — UNANNOTATED stdlib-time alias in field position.
	Silent SilentTime `json:"silent"`
}
