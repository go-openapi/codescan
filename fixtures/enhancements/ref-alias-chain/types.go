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

// Datestamp aliases any but the user provides a swagger:strfmt
// annotation expecting the alias to surface as `{string, date}`.
// Q13 witness: does the classifier-side override fire before the
// any-recognizer empties the schema?
//
// swagger:model Datestamp
// swagger:strfmt date
type Datestamp = any

// Loose aliases any with NO swagger:model annotation. Reachable
// only via Envelope.Drift below. Q13 witness for the unannotated
// case: under R6 the use-site dispatch may or may not produce a
// definition for the alias depending on whether the recognizer
// fires before the annotation gate.
type Loose = any

// UserIDStrf probes whether `swagger:strfmt` on an alias of a
// primitive is honoured. Expected user intent: the alias surfaces
// as `{string, uuid}`. Q13-adjacent witness; tells us whether the
// classifier-not-consulted bug also affects alias-of-primitive
// (it does, pre-patch).
//
// swagger:model UserIDStrf
// swagger:strfmt uuid
type UserIDStrf = int64

// CountTyped probes the `swagger:type` override on an alias of
// any. `swagger:type` takes a Go-type name (per
// `SwaggerSchemaForType`); IS consulted today via
// `classifierNamedTypeOverride`. Baseline to confirm the patch
// preserves the existing handling.
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

	// Drift uses an unannotated alias of any — R6 should dissolve
	// it at the field site.
	Drift Loose `json:"drift"`
}
