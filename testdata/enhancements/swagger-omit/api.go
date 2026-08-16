// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package swaggeromit exercises `swagger:omit` — the author's escape hatch for an embed that
// promotes more than the API should carry (go-swagger#1992).
//
// The annotation is a PRE-filter: the listed Go field names are never promoted, so the same
// annotation reads identically whether the embed is inlined or composed into an allOf member.
package swaggeromit

// Base is the shared domain type, deliberately free of any swagger annotation — the whole point of
// the idiom is that the author does not own (or does not want to touch) it.
type Base struct {
	ID      int64
	Name    string
	Created string
}

// Inner sits one level below Base, so a qualified target can reach through the embed chain.
type Inner struct {
	Deep    string
	Visible string
}

// Nested embeds Inner, so Deep is promoted twice over into anything embedding Nested.
type Nested struct {
	Inner

	Shallow string
}

// EmbedLevel uses the ergonomic form: the annotation sits on the embed, and the targets are plain
// field names of that type.
//
// swagger:model EmbedLevel
type EmbedLevel struct {
	// swagger:omit ID,Created
	Base

	Extra string `json:"extra"`
}

// DeclLevel uses the power form on the declaration: a dotted path names the embed chain, a bare
// name resolves against the promoted set.
//
// swagger:model DeclLevel
// swagger:omit Base.ID,Created
type DeclLevel struct {
	Base

	Extra string `json:"extra"`
}

// DeepPath reaches through two embeds with a qualified path, and drops a field promoted from the
// nested level.
//
// swagger:model DeepPath
// swagger:omit Nested.Deep
type DeepPath struct {
	Nested

	Extra string `json:"extra"`
}

// Decorated is the override case: the outer field wins in Go, and omitting the promoted twin keeps
// the allOf rendering from carrying both.
//
// swagger:model Decorated
type Decorated struct {
	// swagger:omit ID
	Base

	// ID is assigned by the server.
	//
	// read only: true
	ID int64
}

// Retyped is the override case that produces an unsatisfiable allOf without the omission: integer
// AND string for the same property.
//
// swagger:model Retyped
type Retyped struct {
	// swagger:omit ID
	Base

	ID string
}

// Unresolved names a field that does not exist — the typo case, reported as a Hint and otherwise
// ignored.
//
// swagger:model Unresolved
type Unresolved struct {
	// swagger:omit Createed
	Base

	Extra string `json:"extra"`
}

// Shadowed re-declares a promoted field with `json:"-"`, which does NOT hide it in Go — the Hint
// points at swagger:omit.
//
// swagger:model Shadowed
type Shadowed struct {
	Base

	Created string `json:"-"`
}

// ModelBase is an annotated model, so composing it into an allOf member emits a $ref.
//
// swagger:model ModelBase
type ModelBase struct {
	A string `json:"a"`
	B string `json:"b"`
}

// BehindRef composes an annotated model, where the omission cannot be expressed: Swagger 2.0 has no
// way to subtract a property from a $ref, so it is dropped with a Hint.
//
// swagger:model BehindRef
type BehindRef struct {
	// swagger:allOf
	// swagger:omit B
	ModelBase

	Extra string `json:"extra"`
}

// CreateThing is the go-swagger#1992 shape verbatim: an inline body struct embedding a shared type
// the author does not want to annotate. The parameters builder walks the body through the schema
// builder, so the embed-level annotation needs no separate plumbing.
//
// swagger:parameters createThing
type CreateThing struct {
	// in: body
	Body struct {
		// swagger:omit ID,Created
		Base
	}
}

// swagger:route POST /things things createThing
//
// Creates a thing.
//
// responses:
//
//	200: omitResp

// omitResp carries the models so they are reachable without ScanModels.
//
// swagger:response omitResp
type omitResp struct {
	// in: body
	Body struct {
		EmbedLevel EmbedLevel `json:"embedLevel"`
		DeclLevel  DeclLevel  `json:"declLevel"`
		DeepPath   DeepPath   `json:"deepPath"`
		Decorated  Decorated  `json:"decorated"`
		Retyped    Retyped    `json:"retyped"`
		Unresolved Unresolved `json:"unresolved"`
		Shadowed   Shadowed   `json:"shadowed"`
		BehindRef  BehindRef  `json:"behindRef"`
	}
}

// swagger:route GET /omits omits listOmits
//
// Lists the omit shapes.
//
// responses:
//
//	200: omitResp
