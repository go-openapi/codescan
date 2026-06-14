// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package bug2701 reproduces go-swagger issue #2701 ("In path parameter for an
// embedded struct is ignored and thus defaults to query"): an `in: path`
// annotation on an EMBEDDED struct field in a swagger:parameters type is
// ignored — the promoted fields default to `in: query`.
//
// The fixture also guards the exportedness contract: only EXPORTED fields are
// promoted (the product documents the public API surface), including exported
// fields reached recursively through an unexported embedded type
// (Go promotes them — outer.Trace is accessible). Unexported fields, at any
// depth, must never surface as parameters.
package bug2701

// auditInfo is an UNEXPORTED struct embedded (recursively) inside URCapID. Its
// exported field must still be promoted — Go promotes it and the field is
// reachable on the outer type, so it is part of the API — while its unexported
// field must never be rendered.
type auditInfo struct {
	Trace string `json:"trace"` // exported → promoted recursively, inherits in:path
	token string                // unexported → never rendered
}

type URCapID struct {
	Vendor    string `json:"vendorID"`
	ID        string `json:"urcapID"`
	Version   string `json:"version"`
	secret    string // unexported → never rendered
	auditInfo        // nested unexported embed
}

// swagger:parameters deleteURCap
type urcapIDParam struct {
	// in: path
	// required: true
	URCapID
}

// swagger:route DELETE /urcap/{vendorID}/{urcapID}/{version}/{trace} URCap deleteURCap
//
// responses:
//   201: description: ok
func h() {}
