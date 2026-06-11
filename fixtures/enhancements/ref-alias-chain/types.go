// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package ref_alias_chain exercises buildDeclAlias under RefAliases=true
// with an alias-of-alias chain and aliases to well-known standard library
// types (time.Time, any).
package ref_alias_chain

import "time"

// BaseBody is the concrete named struct at the bottom of the alias chain.
//
// swagger:model BaseBody
type BaseBody struct {
	// required: true
	ID int64 `json:"id"`

	Name string `json:"name"`
}

// LinkA is a direct alias of BaseBody.
//
// swagger:model LinkA
type LinkA = BaseBody

// LinkB is an alias of an alias — chain depth two.
//
// swagger:model LinkB
type LinkB = LinkA

// Timestamp aliases time.Time so that buildDeclAlias takes its isStdTime
// branch.
//
// swagger:model Timestamp
type Timestamp = time.Time

// Wildcard aliases any so that buildDeclAlias takes its isAny branch.
//
// swagger:model Wildcard
type Wildcard = any

// Datestamp aliases any with a user-provided `swagger:strfmt`
// annotation. Expected: the alias surfaces as `{string, date}` —
// the decl-entry strfmt classifier fires before the any-recognizer
// would otherwise empty the schema.
//
// swagger:model Datestamp
// swagger:strfmt date
type Datestamp = any

// Loose aliases any with NO swagger:model annotation. Reachable
// only via Envelope.Drift below. Witness for the unannotated case
// at field sites: the alias dissolves and produces no definition.
type Loose = any

// UserIDStrf probes `swagger:strfmt` on an alias of a primitive.
// Expected: the alias surfaces as `{string, uuid}` — the
// decl-entry strfmt classifier wins over the primitive underlying.
//
// swagger:model UserIDStrf
// swagger:strfmt uuid
type UserIDStrf = int64

// CountTyped probes the `swagger:type` override on an alias of
// any. `swagger:type` takes a Go-type name (per
// `SwaggerSchemaForType`); honoured via
// `classifierNamedTypeOverride`.
//
// swagger:model CountTyped
// swagger:type int64
type CountTyped = any

// Envelope references the aliases via its fields so the scanner also
// walks the schemaBuilder.buildAlias path for each chain member.
//
// swagger:model Envelope
type Envelope struct {
	First LinkA `json:"first"`

	Second LinkB `json:"second"`

	CreatedAt Timestamp `json:"createdAt"`

	Meta Wildcard `json:"meta"`

	// Stamp is typed via the strfmt-annotated alias of `any`.
	Stamp Datestamp `json:"stamp"`

	// Drift uses an unannotated alias of any — the field site
	// dissolves the alias to the underlying.
	Drift Loose `json:"drift"`
}
