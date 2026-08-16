// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_responses_calibration exercises the responses
// builder's alias handling across three reach contexts:
//
//  1. Top-level alias annotated `swagger:response` —
//     AliasedTopResponse = internalResponse.
//  2. Body field typed as an alias, both annotated and unannotated
//     — BodyAliasPlainResponse / BodyAliasModeledResponse.
//  3. Non-body (header) field typed as an alias —
//     DirectResponse.XHeaderPlain.
//
// `DirectResponse` carries the control case (BodyDirect typed as
// the raw Envelope model) alongside the alias variants so the
// comparison is on one canvas. Two real `swagger:route` handlers
// bind the responses to operations so the response shapes are
// visible in the captured spec under the operations' `responses`
// block.
//
// See the responses builder's alias-handling contract:
// [§alias-handling](../../../internal/builders/responses/README.md#alias-handling).
package alias_responses_calibration

// Envelope is the canonical response body model.
//
// swagger:model Envelope
type Envelope struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// HeaderID is a named string backing the non-body (header) alias
// witnesses. SimpleSchema targets can't carry $ref, so non-body
// aliases always expand regardless of annotation.
type HeaderID string

// EnvelopeAlias is an UNANNOTATED alias of Envelope. At body field
// sites it dissolves to `$ref: Envelope` (not `$ref: EnvelopeAlias`),
// and does not produce its own `definitions` entry.
type EnvelopeAlias = Envelope

// EnvelopeAliasModeled is the ANNOTATED counterpart of
// EnvelopeAlias. `swagger:model` opts the alias in as a
// first-class spec entity — body field sites preserve
// `$ref: EnvelopeAliasModeled` and the alias gets its own
// definition.
//
// swagger:model EnvelopeAliasModeled
type EnvelopeAliasModeled = Envelope

// EnvelopeAlias2 is a 2-link unannotated chain. Both links
// dissolve → body fields land on `$ref: Envelope`.
type EnvelopeAlias2 = EnvelopeAlias

// HeaderIDAlias is an unannotated alias of HeaderID. Non-body
// SimpleSchema use sites can't carry $ref regardless of annotation;
// the witness mainly asserts the alias does not produce a dangling
// definition.
type HeaderIDAlias = HeaderID

// internalResponse is the unexported backing struct for the
// top-level aliased response below. The contract: this backing
// struct must not surface as a `definitions` entry even though
// nothing else annotates it.
type internalResponse struct {
	// Body is the response body for the aliased top-level response.
	//
	// in: body
	Body Envelope `json:"body"`

	// XSession is a header on the aliased top-level response.
	//
	// in: header
	XSession HeaderID `json:"X-Session"`
}

// AliasedTopResponse is a top-level alias annotated
// `swagger:response`. The responses builder is transparent re:
// model creation at this layer — neither this alias nor its
// backing struct surfaces in `definitions`. The response's body /
// headers come from the fields of the unaliased target.
//
// swagger:response aliasedTopResponse
type AliasedTopResponse = internalResponse

// DirectResponse is the control response: declared directly (not
// via an alias), holding body and non-body fields typed as the
// alias variants above so every reach context is visible inside
// one response.
//
// swagger:response directResponse
type DirectResponse struct {
	// BodyDirect — body typed as the raw Envelope model (control
	// case, alias-handling-independent).
	//
	// in: body
	BodyDirect Envelope `json:"bodyDirect"`

	// XHeaderPlain — header typed as the UNANNOTATED alias
	// HeaderIDAlias. SimpleSchema target — the alias must expand
	// to its primitive type inline regardless of mode, and
	// HeaderIDAlias must not appear in `definitions`.
	//
	// in: header
	XHeaderPlain HeaderIDAlias `json:"X-Header-Plain"`
}

// BodyAliasPlainResponse — body response typed as the UNANNOTATED
// alias EnvelopeAlias. The body schema's $ref dissolves to
// Envelope.
//
// swagger:response bodyAliasPlainResponse
type BodyAliasPlainResponse struct {
	// in: body
	Body EnvelopeAlias `json:"body"`
}

// BodyAliasModeledResponse — body response typed as the
// ANNOTATED alias EnvelopeAliasModeled. The body schema's $ref
// preserves EnvelopeAliasModeled (and the alias surfaces as its
// own definition).
//
// swagger:response bodyAliasModeledResponse
type BodyAliasModeledResponse struct {
	// in: body
	Body EnvelopeAliasModeled `json:"body"`
}

// BodyAliasChainResponse — body response typed as the 2-link
// unannotated chain EnvelopeAlias2. Both layers dissolve to
// Envelope.
//
// swagger:response bodyAliasChainResponse
type BodyAliasChainResponse struct {
	// in: body
	Body EnvelopeAlias2 `json:"body"`
}
