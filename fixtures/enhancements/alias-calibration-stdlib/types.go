// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_calibration_stdlib is the cycle-2 calibration fixture
// for the W3 alias workshop. Same structural pattern as cycle 1
// (annotated decls + Envelope using them as fields + one unannotated
// case for the R4 confirmation), but with stdlib-special RHS types
// instead of primitives.
//
// `applyStdlibSpecials` recognizes time.Time, error, json.RawMessage,
// and any. The schema builder hits this recognizer in some code paths
// but not all — Q3 is the bug that surfaces when `buildDeclAlias`'s
// Expand branch walks `Underlying()` without checking specials first.
//
// Cells exercised (per mode):
//
//   - decl-as-model × RHS=stdlib(time.Time) → Timestamp
//   - decl-as-model × RHS=stdlib(error) → Err
//   - decl-as-model × RHS=stdlib(json.RawMessage) → Raw
//   - decl × RHS=stdlib(time.Time) × UNANNOTATED → SilentTime
//   - field × RHS=stdlib(time.Time) via annotated alias → Envelope.at
//   - field × RHS=stdlib(error) via annotated alias → Envelope.failure
//   - field × RHS=stdlib(json.RawMessage) via annotated alias → Envelope.payload
//   - field × RHS=stdlib(time.Time) via unannotated alias → Envelope.silent
//
// 8 cells × 3 modes = 24 judgments. Designed to test where R1
// (modes collapse for primitive RHS) breaks: stdlib-special RHS
// produces *types.Named (not *types.Basic), so the $ref-branch
// switch in schema.go:174 has a matching case — Default and Ref
// may diverge.
//
// See `.claude/plans/workshops/alias-matrix.md` and
// `.claude/plans/workshops/alias-ledger.md` cycle 2.
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
// annotation — reachable only via Envelope.Silent. Tests whether
// R4 ("annotation gates definitions") holds for stdlib-special RHS
// the way it did for primitive RHS in cycle 1.
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
