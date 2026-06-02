// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package header_named_basic pins the M1-follow-up fix that brought
// `in: header` parameters into the SimpleSchema-aware primitive-
// inline path. Pre-fix, classifierNamedBasic only short-circuited
// to inline-emission for query / path / formData (the v1
// `isAliasParam` predicate omitted `header`), so a `type SessionID
// string` field appearing as `in: header` silently fell through to
// FindModel → makeRef and emitted a `$ref`-typed parameter — invalid
// under OAS v2 SimpleSchema.
//
// With the fix, all four non-body locations route through the same
// SimpleSchema-driven primitive-inline arm and the header parameter
// emits `{type: string, format: ""}` inline.
package header_named_basic

// SessionID is a named string used as a header parameter. The
// schema builder used to emit a $ref to a top-level definition for
// it; under SimpleSchema mode for `in: header` it now inlines as
// `{string, ""}`.
type SessionID string

// HeaderParams demonstrates a header parameter typed as a named
// string.
//
// swagger:parameters headerNamedBasicOp
type HeaderParams struct {
	// Session is the offending parameter — pre-fix it emitted as a
	// $ref to definitions/SessionID, post-fix it inlines as a
	// primitive string.
	//
	// in: header
	Session SessionID `json:"X-Session"`
}

// DoHeader handles the route.
//
// swagger:route GET /header-named-basic hdr headerNamedBasicOp
//
// Responses:
//
//	200: description: OK
func DoHeader() {}
