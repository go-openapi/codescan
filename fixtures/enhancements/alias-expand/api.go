// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_expand exercises non-transparent alias handling for both
// parameter and response scanners. It is designed to be scanned with the
// default Options (TransparentAliases=false) and with RefAliases=true to
// cover the two non-transparent code paths.
package alias_expand

// Payload is the canonical struct referenced by aliases.
//
// swagger:model Payload
type Payload struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// PayloadAlias aliases Payload once.
type PayloadAlias = Payload

// PayloadAlias2 aliases PayloadAlias (alias-of-alias chain).
type PayloadAlias2 = PayloadAlias

// QueryID is a named string used as the base of a non-body parameter alias.
type QueryID string

// QueryIDAlias aliases QueryID for a non-body parameter field.
type QueryIDAlias = QueryID

// AliasedParams exposes one body parameter that is an alias, one body
// parameter that is an alias of an alias, and one non-body parameter that
// is an alias of a primitive-backed named type.
//
// swagger:parameters aliasedRequest
type AliasedParams struct {
	// BodyPrimary is a body parameter of aliased struct type.
	//
	// in: body
	// required: true
	BodyPrimary PayloadAlias `json:"bodyPrimary"`

	// BodyNested is a body parameter whose type is an alias of an alias.
	//
	// in: body
	BodyNested PayloadAlias2 `json:"bodyNested"`

	// Lookup is a query parameter aliased off a named primitive type.
	//
	// in: query
	Lookup QueryIDAlias `json:"lookup"`
}

// ResponseEnvelope is the canonical struct referenced by aliases used in
// responses. Its `payload` field is typed as the UNannotated `PayloadAlias`,
// so the field's $ref dissolves to `Payload` (annotation gate at use sites).
//
// swagger:model ResponseEnvelope
type ResponseEnvelope struct {
	Payload PayloadAlias `json:"payload"`
}

// PayloadAliasModeled is the annotated counterpart of `PayloadAlias`.
// Same underlying Go type, but `swagger:model` opts the alias in as a
// first-class spec entity.
//
// swagger:model PayloadAliasModeled
type PayloadAliasModeled = Payload

// ResponseEnvelopeModeled is the bidirectional sibling of
// `ResponseEnvelope`. Same struct shape, but its `payload` field uses
// the annotated alias `PayloadAliasModeled`. The field's $ref preserves
// the alias identity (`$ref: PayloadAliasModeled`), demonstrating that
// the explicit `swagger:model` opt-in surfaces the alias name at body
// field sites.
//
// swagger:model ResponseEnvelopeModeled
type ResponseEnvelopeModeled struct {
	Payload PayloadAliasModeled `json:"payload"`
}

// EnvelopeAlias aliases ResponseEnvelope once.
type EnvelopeAlias = ResponseEnvelope

// EnvelopeAlias2 aliases EnvelopeAlias (alias-of-alias).
type EnvelopeAlias2 = EnvelopeAlias

// AliasedResponse has a body field whose type is an alias chain.
//
// swagger:response aliasedResponse
type AliasedResponse struct {
	// Body is an alias of an alias.
	//
	// in: body
	Body EnvelopeAlias2 `json:"body"`
}

// EnvelopeAliasModeled is the annotated counterpart of
// EnvelopeAlias. With `swagger:model` the alias opts in as a
// first-class spec entity; the alias name is preserved at body
// field sites in `swagger:response` structs.
//
// swagger:model EnvelopeAliasModeled
type EnvelopeAliasModeled = ResponseEnvelope

// AliasedModeledResponse is the bidirectional sibling of
// AliasedResponse: same shape, but its Body field uses the
// ANNOTATED alias `EnvelopeAliasModeled`. The body schema's
// `$ref` preserves `EnvelopeAliasModeled` (alongside the alias's
// own definition), while AliasedResponse dissolves to
// `$ref: ResponseEnvelope`. Both halves on one canvas.
//
// swagger:response aliasedModeledResponse
type AliasedModeledResponse struct {
	// Body is an annotated alias.
	//
	// in: body
	Body EnvelopeAliasModeled `json:"body"`
}

// exportedParams is the backing struct for an aliased swagger:parameters.
type exportedParams struct {
	// in: query
	Search string `json:"search"`

	// in: body
	// required: true
	Data Payload `json:"data"`
}

// AliasedTopParams annotates an alias as the parameters set: the scanner
// must resolve the alias via parameterBuilder.buildAlias.
//
// swagger:parameters aliasedTop
type AliasedTopParams = exportedParams

// AliasedTopParams2 chains AliasedTopParams through a second alias layer.
//
// swagger:parameters aliasedTop2
type AliasedTopParams2 = AliasedTopParams

// NamedTopResponse is a plain struct annotated as a response — used to
// keep a deterministic response in the expand-mode fixture even though
// response-level aliasing is deferred to the alias-response fixture.
//
// swagger:response namedTopResponse
type NamedTopResponse struct {
	// in: body
	Body ResponseEnvelope `json:"body"`
}
