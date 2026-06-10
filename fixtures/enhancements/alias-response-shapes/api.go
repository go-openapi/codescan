// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_response_shapes captures Q4-exploration goldens
// for the response builder's alias handling on MEMBER fields:
//
//   - body field whose Go type is an alias chain
//   - header field whose Go type is an alias of a named primitive
//   - header field whose Go type is an alias of a named struct
//
// Top-level response-as-alias is intentionally NOT in this package
// because the default (non-Transparent, non-Ref) buildAlias path
// for swagger:response on an alias crashes with "anonymous types
// are currently not supported for responses" (the existing
// alias-response/ fixture documents that gap by running only under
// RefAliases=true). The split lets default mode run on this
// package — the member-alias cases work — and the goldens
// captured side-by-side reveal what's broken vs working.
package alias_response_shapes

// Envelope is the canonical named struct.
//
// swagger:model Envelope
type Envelope struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// EnvelopeAlias is an alias of Envelope.
type EnvelopeAlias = Envelope

// EnvelopeAlias2 is an alias of EnvelopeAlias (alias-of-alias).
type EnvelopeAlias2 = EnvelopeAlias

// SessionID is a named string used in alias scenarios.
type SessionID string

// SessionIDAlias is an alias of SessionID.
type SessionIDAlias = SessionID

// BodyAliasResponse — body field uses an alias-of-alias chain.
//
// swagger:response bodyAliasResponse
type BodyAliasResponse struct {
	// in: body
	Body EnvelopeAlias2 `json:"body"`
}

// HeaderAliasedBasicResponse — header field is an alias of a
// named string. Expected emission: primitive inline
// {string, ""}; no $ref under any mode (headers can't carry
// $ref).
//
// swagger:response headerAliasedBasicResponse
type HeaderAliasedBasicResponse struct {
	// in: header
	Session SessionIDAlias `json:"X-Session"`
}

// HeaderAliasedStructResponse — header field is an alias of a
// named struct. Expected emission: NO body schema corruption
// (Q2 fix); header surfaces empty (struct can't reduce to a
// SimpleSchema primitive); CodeUnsupportedInSimpleSchema
// diagnostic fires for the underlying ref attempt.
//
// swagger:response headerAliasedStructResponse
type HeaderAliasedStructResponse struct {
	// in: header
	Detail EnvelopeAlias `json:"X-Detail"`
}

// EnvelopeAliasModeled is the annotated counterpart of
// EnvelopeAlias. R8 makes the annotation the gate for first-class
// alias identity at body field sites; this decl exists so the
// bidirectional contract is visible on the same canvas as the
// unannotated witnesses above.
//
// swagger:model EnvelopeAliasModeled
type EnvelopeAliasModeled = Envelope

// BodyAliasModeledResponse — body field uses the ANNOTATED alias
// EnvelopeAliasModeled. Under R8 the body schema's `$ref`
// preserves the alias name (`$ref: EnvelopeAliasModeled`),
// recovering the pre-R8 alias-name $ref behaviour for users who
// explicitly opt in via `swagger:model`. The unannotated
// counterpart (BodyAliasResponse above) dissolves to
// `$ref: Envelope`.
//
// swagger:response bodyAliasModeledResponse
type BodyAliasModeledResponse struct {
	// in: body
	Body EnvelopeAliasModeled `json:"body"`
}
