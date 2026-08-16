// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package alias_response exercises response-level top-alias handling.
// It is scanned with RefAliases=true because the non-transparent expand
// path on top-level response aliases is not supported by the scanner.
package alias_response

// Envelope is the canonical response body type.
//
// swagger:model Envelope
type Envelope struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// exportedResponse is the backing struct for the aliased response.
type exportedResponse struct {
	// in: body
	Body Envelope `json:"body"`
}

// AliasedTopResponse annotates an alias as the response: the scanner
// resolves it via responseBuilder.buildAlias under RefAliases=true.
//
// swagger:response aliasedTopResponse
type AliasedTopResponse = exportedResponse

// AliasedTopResponse2 chains AliasedTopResponse through a second alias.
//
// swagger:response aliasedTopResponse2
type AliasedTopResponse2 = AliasedTopResponse
