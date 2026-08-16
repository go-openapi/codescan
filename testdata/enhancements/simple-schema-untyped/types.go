// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package simple_schema_untyped witnesses the types that say nothing, in the two positions where
// saying nothing is not an option.
//
// OAS v2 makes `type` required on every non-body parameter and on every response header, so an empty
// schema is not a permissive answer there — it is an invalid one. `any` and `json.RawMessage` are the
// usual sources: "any JSON" is a real answer for a body or a definition, and no answer at all outside
// them. Both used to emit `{}` and no diagnostic.
//
// The fallback is `{type: string}`, with no format invented. A query string, a path segment and a
// header are percent-encoded text, so `binary` would claim octets the position cannot carry and
// `byte` would claim a base64 framing nobody applied. `string` claims only that something textual
// arrives, which is all that is known.
//
// The same two fields in a BODY keep the empty schema: there the answer is legal and correct.
//
// See [§simple-schema-mode](../../../internal/builders/schema/README.md#simple-schema-mode).
package simple_schema_untyped

import "encoding/json"

// Payload carries the same two fields into a body, where empty stays empty.
type Payload struct {
	Anything any             `json:"anything"`
	Raw      json.RawMessage `json:"raw"`
}

// UntypedParams reaches the underspecified types from a non-body parameter.
//
// swagger:parameters getUntyped
type UntypedParams struct {
	// Anything says nothing at all.
	//
	// in: query
	Anything any `json:"anything"`

	// Raw says "some JSON", which a query string cannot carry as anything but text.
	//
	// in: query
	Raw json.RawMessage `json:"raw"`

	// Body is the control: the very same types are legal as an empty schema here.
	//
	// in: body
	Body Payload `json:"body"`
}

// UntypedResponse reaches them from a response header, and keeps the body control.
//
// swagger:response untypedResponse
type UntypedResponse struct {
	// Anything is a header that resolves to nothing.
	Anything any `json:"X-Anything"`

	// Raw is a header carrying opaque JSON.
	Raw json.RawMessage `json:"X-Raw"`

	// in: body
	Body Payload `json:"body"`
}
