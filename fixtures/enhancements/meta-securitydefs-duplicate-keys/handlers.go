// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package meta_securitydefs_duplicate_keys witnesses two intertwined
// behaviours of the swagger:meta SecurityDefinitions raw block.
//
// 1. Grammar lexer: every body line MUST preserve its per-line
// structural indentation — same as extensions / infoExtensions — so
// the YAML nesting under each scheme name survives. Without that,
// both schemes' fields collapse into one flat top-level mapping and
// the spec decode aborts.
//
// 2. yaml sub-parser: duplicate mapping keys at any depth are
// silently last-wins dedupe'd before the strict yaml.v3 decoder
// runs. Here api_key declares "type:" twice on purpose.
//
// Diagnostic emission on duplicates is deferred to the yaml-library
// swap tracked in .claude/plans/forthcoming-features.md §3.1.
//
// swagger:meta
//
// SecurityDefinitions:
//	api_key:
//	     type: apiKey
//	     name: KEY
//	     type: apiKey
//	     in: header
//	oauth2:
//	    type: oauth2
//	    authorizationUrl: /oauth2/auth
//	    tokenUrl: /oauth2/token
//	    in: header
//	    scopes:
//	      bla1: foo1
//	      bla2: foo2
//	    flow: accessCode
package meta_securitydefs_duplicate_keys
