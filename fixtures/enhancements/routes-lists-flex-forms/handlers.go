// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package routes_lists_flex_forms witnesses the unified
// Property.AsList accepted surface forms for list-shaped keywords
// (schemes / consumes / produces). M6.5-D widened KwSchemes from
// asCommaList() to asRawBlock(), added inline-value capture to
// collectRawBlock, and centralised list parsing in Block.GetList
// → Property.AsList. Every form below should produce the same
// underlying token list.
package routes_lists_flex_forms

// CommaInline swagger:route GET /alpha lists commaInline
//
// Inline comma-separated form.
//
// Schemes: http, https
// Consumes: application/json
// Produces: application/json
//
// Responses:
//
//	200: description: OK
func CommaInline() {}

// YAMLDash swagger:route GET /beta lists yamlDash
//
// Multi-line YAML-dash form.
//
// Schemes:
//   - http
//   - https
//
// Consumes:
//   - application/json
//   - application/xml
//
// Produces:
//   - application/json
//
// Responses:
//
//	200: description: OK
func YAMLDash() {}

// BareLines swagger:route GET /gamma lists bareLines
//
// Multi-line indented bare-lines form (no `-` markers).
//
// Consumes:
//
//	application/json
//	application/xml
//
// Produces:
//
//	application/json
//
// Responses:
//
//	200: description: OK
func BareLines() {}

// MixedInlineAndYAML swagger:route GET /delta lists mixedInlineYAML
//
// Inline-plus-indented continuation.
//
// Schemes: http
//   - https
//   - ws
//
// Responses:
//
//	200: description: OK
func MixedInlineAndYAML() {}
