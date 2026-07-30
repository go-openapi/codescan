// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package defaultallofembedsoverride is the minimal repro for what happens to a
// field-override embed under Options.DefaultAllOfForEmbeds.
//
// The override idiom (go-swagger#1992): a struct embeds a shared, annotation-free
// domain type and RE-DECLARES one of the promoted fields — either to decorate it
// (`read only: true`, description, validations) or to drop it from the wire with
// `json:"-"`. Go resolves this by depth: the outer field shadows the promoted one,
// so exactly one field of that name exists at the shallowest depth.
//
// Inlined (the default), the schema builder mirrors that: the re-declaration wins
// and one property is emitted. Composed into `allOf` by DefaultAllOfForEmbeds, the
// two declarations land in DIFFERENT members, and neither Go's depth rule nor the
// builder's override resolution can reach across a member boundary.
package defaultallofembedsoverride

// Base is the shared domain type. It carries no swagger annotations at all — the
// point of the idiom is to leave it alone.
type Base struct {
	ID      int64
	Name    string
	Created string
}

// Decorated re-declares a promoted field to DECORATE it, leaving it on the wire.
//
// swagger:model Decorated
type Decorated struct {
	Base

	// ID is assigned by the server.
	//
	// read only: true
	ID int64
}

// Muted re-declares a promoted field to drop it from the wire.
//
// swagger:model Muted
type Muted struct {
	Base

	Created string `json:"-"`
}

// Renamed re-declares a promoted field under a different JSON name.
//
// swagger:model Renamed
type Renamed struct {
	Base

	Name string `json:"fullName"`
}

// Retyped re-declares a promoted field with a DIFFERENT Go type — legal Go, and
// the outer declaration is the only one that marshals.
//
// swagger:model Retyped
type Retyped struct {
	Base

	ID string
}

// respBody carries the models so they are reachable without ScanModels.
//
// swagger:response overrideResp
type respBody struct {
	// in: body
	Body struct {
		Decorated Decorated `json:"decorated"`
		Muted     Muted     `json:"muted"`
		Renamed   Renamed   `json:"renamed"`
		Retyped   Retyped   `json:"retyped"`
	}
}

// swagger:route GET /overrides overrides listOverrides
//
// Lists override shapes.
//
// responses:
//
//	200: overrideResp
