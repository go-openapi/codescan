// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package meta_tos_schemes_terminator pins the sibling raw-block
// terminator rule for `Terms Of Service:` (KwTOS) when immediately
// followed by `Schemes:` (KwSchemes). Both keywords are asRawBlock
// since M6.5-D widened KwSchemes from asCommaList; both share the
// CtxMeta family, so the family-based terminator rule must fire
// (lexer.go: isSiblingTerminatorFor → familyOf overlap on CtxMeta).
//
// Q26 was originally pinned when a pre-merge state of the genspec-tui
// branch observed the petstore meta producing
// `"termsOfService": "url\nSchemes:\nhttp"` — Schemes absorbed into
// TOS body. Building this witness during the fix-quirks A2 pass
// revealed the behaviour is now CORRECT on master (verified directly
// against the upstream go-swagger generated source). Some commit
// during Stream M's merge sequence closed it; we never identified
// which. This fixture locks the corrected behaviour as a
// regression-detector golden so any future change to the terminator
// rule or to KwSchemes shape will turn this red.
//
// Source shape mirrors go-swagger's generated meta exactly:
// tab-indented overall, 2-space body indent under each keyword, no
// blank line between TOS body and Schemes.
//
// swagger:meta
//
//	Terms Of Service:
//	  http://example.com/terms/
//	Schemes:
//	  http
//	  https
package meta_tos_schemes_terminator
