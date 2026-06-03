// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_calibration_primitive is the cycle-1 calibration
// fixture for the W3 alias workshop. It is intentionally the smallest
// possible alias-decl scenario, designed to surface the annotation-vs-
// auto-discovery distinction Fred surfaced in cycle 1.
//
// Three annotated aliases (UserID / Name / Active) sit alongside two
// unannotated aliases (LegacyID / LegacyNick) — same primitive RHS,
// only the swagger:model annotation differs. The Envelope struct
// exposes both flavours as fields so the comparison is visible in
// one spec output per mode.
//
// Cells exercised (per mode):
//
//   - decl-as-model × RHS=Basic × annotated → UserID, Name, Active
//   - field × RHS=Basic via annotated alias → Envelope.user, .nick, .on
//   - decl × RHS=Basic × UNANNOTATED → LegacyID, LegacyNick
//   - field × RHS=Basic via unannotated alias → Envelope.legacy, .legacyNick
//
// 10 cells × 3 modes = 30 judgments. The annotated/unannotated pair
// directly tests Fred's rule: "swagger:model forces a definition + $ref;
// auto-discovered should not produce a dangling definition."
//
// See `.claude/plans/workshops/alias-matrix.md` §6 and
// `.claude/plans/workshops/alias-ledger.md` for the judgment record.
package alias_calibration_primitive

// UserID is an alias to int64 exposed as a top-level model.
//
// swagger:model UserID
type UserID = int64

// Name is an alias to string exposed as a top-level model.
//
// swagger:model Name
type Name = string

// Active is an alias to bool exposed as a top-level model.
//
// swagger:model Active
type Active = bool

// LegacyID is an alias to int64 with NO swagger:model annotation —
// reachable only via Envelope.Legacy. Tests whether auto-discovered
// aliases produce a (possibly dangling) definition or are dissolved
// at the use site.
type LegacyID = int64

// LegacyName is an alias to string with NO swagger:model annotation —
// reachable only via Envelope.LegacyNick.
type LegacyName = string

// Envelope uses both annotated and unannotated primitive aliases as
// field types. Exercises the field-reach context for each, and the
// annotation-vs-auto-discovery distinction.
//
// swagger:model Envelope
type Envelope struct {
	// User identifier — annotated alias-of-int64 in field position.
	User UserID `json:"user"`

	// Nick — annotated alias-of-string in field position.
	Nick Name `json:"nick"`

	// On — annotated alias-of-bool in field position.
	On Active `json:"on"`

	// Legacy — UNANNOTATED alias-of-int64 in field position.
	Legacy LegacyID `json:"legacy"`

	// LegacyNick — UNANNOTATED alias-of-string in field position.
	LegacyNick LegacyName `json:"legacyNick"`
}
