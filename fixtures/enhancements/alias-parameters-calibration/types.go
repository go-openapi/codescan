// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_parameters_calibration is the cycle-4 calibration
// fixture for the W3 alias workshop. It is the parameters-builder
// analogue of cycle-3 (alias-calibration-embed).
//
// The fixture exists to surface the parameters-builder's current
// behaviour for the three reach contexts the cycle-4 workshop
// has to cover:
//
//  1. Top-level alias annotated `swagger:parameters` —
//     AliasedTopParams = internalParams.
//  2. Body field typed as an alias, both annotated and unannotated —
//     DirectParams.BodyAliasPlain / .BodyAliasModeled.
//  3. Non-body field typed as an alias — DirectParams.LookupPlain.
//
// `DirectParams` carries the control case (BodyDirect typed as the
// raw Payload model) alongside the alias variants so the comparison
// is on one canvas.
//
// Two real `swagger:route` handlers bind the parameter sets to
// operations so `paths` populates in the captured spec — without
// them the parameters sets would be registered but never merged
// into an operation, and the body/query semantics would not be
// visible in the goldens.
//
// 9 decls × 3 modes = 27 base cells. The Q12 leak — unexported
// backing struct surfacing in `definitions` — should be visible
// in the pre-patch captures under all three modes.
//
// See `.claude/plans/workshops/alias-parameters.md` §4 for the
// rule candidate (R7) and `.claude/plans/workshops/alias-ledger.md`
// cycle 4 for the running judgment record.
package alias_parameters_calibration

// Payload is the canonical body model.
//
// swagger:model Payload
type Payload struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// QueryID is a named string backing the non-body alias witnesses.
type QueryID string

// PayloadAlias is an UNANNOTATED alias of Payload. Under R7 / R6
// it must dissolve at body field sites — `$ref: Payload` rather
// than `$ref: PayloadAlias`, and `PayloadAlias` must not surface
// as its own definition.
type PayloadAlias = Payload

// PayloadAliasModeled is the ANNOTATED counterpart of
// PayloadAlias. The `swagger:model` opt-in makes the alias a
// first-class spec entity — body field sites preserve
// `$ref: PayloadAliasModeled`, and the alias gets its own
// definition.
//
// swagger:model PayloadAliasModeled
type PayloadAliasModeled = Payload

// PayloadAlias2 is a two-link unannotated chain. Both links
// dissolve under R7 → body fields land on `$ref: Payload`.
type PayloadAlias2 = PayloadAlias

// QueryIDAlias is an unannotated alias of QueryID. Non-body
// SimpleSchema use sites can't carry `$ref` regardless of
// annotation, so this case mainly witnesses that the alias does
// not produce a dangling definition.
type QueryIDAlias = QueryID

// internalParams is the unexported backing struct for the
// top-level aliased parameter set below. Q12 witness — pre-R7
// captures surface this struct as a `definitions` entry even
// though it carries no `swagger:model` annotation.
type internalParams struct {
	// Body is a body parameter on the aliased top-level params set.
	//
	// in: body
	// required: true
	Body Payload `json:"body"`

	// Search is a query parameter on the aliased top-level params set.
	//
	// in: query
	Search string `json:"search"`
}

// AliasedTopParams is a top-level alias annotated
// `swagger:parameters`. The R7 clause for top-level decls says:
// neither this alias nor its backing struct should surface in
// `definitions` — the fields of internalParams become the
// `aliasedTop` operation's parameters.
//
// swagger:parameters aliasedTop
type AliasedTopParams = internalParams

// DirectParams is the control parameter set: declared directly
// (not via an alias), holding body and non-body fields typed as
// the alias variants above so every reach context is visible
// inside one struct.
//
// swagger:parameters directParams
type DirectParams struct {
	// BodyDirect — body parameter typed as the raw Payload model
	// (control case, R6/R7-independent).
	//
	// in: body
	// required: true
	BodyDirect Payload `json:"bodyDirect"`

	// BodyAliasPlain — body parameter typed as the UNANNOTATED
	// alias PayloadAlias. Under R7 the body schema's $ref
	// dissolves to Payload.
	//
	// in: body
	BodyAliasPlain PayloadAlias `json:"bodyAliasPlain"`

	// BodyAliasModeled — body parameter typed as the ANNOTATED
	// alias PayloadAliasModeled. Under R7 the body schema's $ref
	// preserves PayloadAliasModeled (and the alias surfaces as
	// its own definition).
	//
	// in: body
	BodyAliasModeled PayloadAliasModeled `json:"bodyAliasModeled"`

	// BodyAliasChain — body parameter typed as the two-link
	// unannotated chain PayloadAlias2. Both layers dissolve to
	// Payload.
	//
	// in: body
	BodyAliasChain PayloadAlias2 `json:"bodyAliasChain"`

	// LookupPlain — query parameter typed as the UNANNOTATED
	// alias QueryIDAlias. SimpleSchema target — the alias must
	// expand to its primitive type inline regardless of mode,
	// and QueryIDAlias must not appear in `definitions`.
	//
	// in: query
	LookupPlain QueryIDAlias `json:"lookupPlain"`
}
