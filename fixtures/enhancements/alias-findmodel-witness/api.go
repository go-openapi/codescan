// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_findmodel_witness exercises the parameters and
// responses builders' GetModel calls on alias targets that are
// not annotated with swagger:model — the FindDecl-fallback path.
//
// Pre-migration FindModel registered such targets in ExtraModels
// as an implicit side effect of the lookup. The explicit GetModel
// + AppendPostDecl pair leaves the registration to the spec
// orchestrator's discovery loop, which visits the queued decl and
// produces the same top-level definition.
//
// The golden captured here locks the two paths to identical
// output, witnessing the safety of the FindModel → GetModel
// migration on the parameters and responses builders.
package alias_findmodel_witness

// PlainTarget is a user struct with no swagger:model annotation.
// It must end up in spec.definitions only via the orchestrator's
// discovery of the alias's RHS — not via any implicit lookup side
// effect at scan time.
type PlainTarget struct {
	// required: true
	ID int64 `json:"id"`

	Note string `json:"note"`
}

// AliasOfPlain is an alias pointing at the unannotated target.
type AliasOfPlain = PlainTarget

// WitnessParams has a body parameter whose Go type is an alias
// of an unannotated struct. Triggers buildFieldAlias on the Body
// field; under RefAliases the GetModel(RHS) lookup at the Named
// switch arm is the relevant call.
//
// swagger:parameters witnessRequest
type WitnessParams struct {
	// in: body
	// required: true
	Body AliasOfPlain `json:"body"`
}

// WitnessResponse has a response body whose Go type is the same
// alias — mirror witness on the response builder's
// buildFieldAlias path.
//
// swagger:response witnessResponse
type WitnessResponse struct {
	// in: body
	Body AliasOfPlain `json:"body"`
}

// AliasOfPlainModeled is the ANNOTATED counterpart of AliasOfPlain.
// Same underlying Named target (PlainTarget — still unannotated by
// design) but the alias itself carries `swagger:model`. Under
// R6 / R7 / R8 the annotation preserves the alias name at every
// use site; without it the alias dissolves to PlainTarget. The
// witness pair below pins both halves on one canvas.
//
// swagger:model AliasOfPlainModeled
type AliasOfPlainModeled = PlainTarget

// WitnessParamsModeled is the bidirectional sibling of WitnessParams.
// Same shape, but its Body field uses the ANNOTATED alias
// `AliasOfPlainModeled`. R7 preserves the alias name in the body
// parameter's $ref while WitnessParams.Body dissolves to PlainTarget.
//
// swagger:parameters witnessModeledRequest
type WitnessParamsModeled struct {
	// in: body
	// required: true
	Body AliasOfPlainModeled `json:"body"`
}

// WitnessResponseModeled mirrors WitnessParamsModeled on the
// response side — R8 preserves the annotated alias's identity at
// the body response schema, recovering the pre-R8 alias-name
// $ref behaviour for opted-in users.
//
// swagger:response witnessModeledResponse
type WitnessResponseModeled struct {
	// in: body
	Body AliasOfPlainModeled `json:"body"`
}
