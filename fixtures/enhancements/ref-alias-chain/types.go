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

// Envelope references the aliases via its fields so the scanner also
// walks the schemaBuilder.buildAlias path for each chain member.
//
// swagger:model Envelope
type Envelope struct {
	First LinkA `json:"first"`

	Second LinkB `json:"second"`

	CreatedAt Timestamp `json:"createdAt"`

	Meta Wildcard `json:"meta"`
}
