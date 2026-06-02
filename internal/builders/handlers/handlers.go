// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package handlers ships shared grammar Walker callbacks for the
// SimpleSchema and full-Schema families of OAS v2 dispatchers.
//
// # Details
//
// See [§dispatch-surface](./README.md#dispatch-surface) for the
// split between SimpleSchema and full-Schema dispatch and
// [§walker-payloads](./README.md#walker-payloads) for the payload
// conventions on each Walker callback.
package handlers

import (
	"strings"

	"github.com/go-openapi/codescan/internal/builders/validations"
	"github.com/go-openapi/codescan/internal/ifaces"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scanner/classify"
	oaispec "github.com/go-openapi/spec"
)

// ExtensionTarget is the minimal surface a Walker.Extension consumer
// needs to write a vendor extension. Implemented by every
// oaispec object that embeds VendorExtensible (Schema, Parameter,
// Header, Response, Operation, …) via the AddExtension method
// promoted from the embed.
type ExtensionTarget interface {
	AddExtension(key string, value any)
}

// Extension returns a Walker.Extension callback that filters non-`x-*`
// names via classify.IsAllowedExtension and writes the typed
// extension value onto target.
//
// # Details
//
// See [§extensions](./README.md#extensions) for the SkipExtensions
// interaction and the wrap-for-side-effects pattern.
func Extension(target ExtensionTarget) func(grammar.Extension) {
	return func(ext grammar.Extension) {
		if !classify.IsAllowedExtension(ext.Name) {
			return
		}
		target.AddExtension(ext.Name, ext.Value)
	}
}

// Number returns a Walker.Number callback that routes
// `maximum:` / `minimum:` / `multipleOf:` onto v.
func Number(v ifaces.ValidationBuilder) func(grammar.Property, float64, bool) {
	return func(pr grammar.Property, val float64, exclusive bool) {
		if !pr.IsTyped() {
			return
		}
		switch pr.Keyword.Name {
		case grammar.KwMaximum:
			v.SetMaximum(val, exclusive)
		case grammar.KwMinimum:
			v.SetMinimum(val, exclusive)
		case grammar.KwMultipleOf:
			v.SetMultipleOf(val)
		}
	}
}

// Integer returns a Walker.Integer callback that routes
// `min/maxLength:` and `min/maxItems:` onto v.
func Integer(v ifaces.ValidationBuilder) func(grammar.Property, int64) {
	return func(pr grammar.Property, val int64) {
		if !pr.IsTyped() {
			return
		}
		switch pr.Keyword.Name {
		case grammar.KwMinLength:
			v.SetMinLength(val)
		case grammar.KwMaxLength:
			v.SetMaxLength(val)
		case grammar.KwMinItems:
			v.SetMinItems(val)
		case grammar.KwMaxItems:
			v.SetMaxItems(val)
		}
	}
}

// UniqueBool returns a Walker.Bool callback that handles only the
// `unique:` keyword. Consumers that also need to dispatch
// `required:` (parameter level) wrap this with a second callback
// via [ComposeBool], or write their own narrow handler that adds the
// parameter-target write next to a call into UniqueBool.
func UniqueBool(v ifaces.ValidationBuilder) func(grammar.Property, bool) {
	return func(pr grammar.Property, val bool) {
		if !pr.IsTyped() {
			return
		}
		if pr.Keyword.Name == grammar.KwUnique {
			v.SetUnique(val)
		}
	}
}

// ComposeBool returns a Walker.Bool callback that fans the payload
// out to every non-nil handler in order. Useful when a consumer
// wants UniqueBool plus a context-specific extra (e.g. parameters'
// `required:` writes to param.Required directly).
func ComposeBool(hs ...func(grammar.Property, bool)) func(grammar.Property, bool) {
	return func(pr grammar.Property, val bool) {
		for _, h := range hs {
			if h != nil {
				h(pr, val)
			}
		}
	}
}

// PatternString returns a Walker.String callback for the `pattern:`
// keyword. The pattern is read from `pr.Value` so the regex source
// reaches v.SetPattern verbatim.
func PatternString(v ifaces.ValidationBuilder) func(grammar.Property, string) {
	return func(pr grammar.Property, _ string) {
		if pr.Keyword.Name == grammar.KwPattern {
			v.SetPattern(pr.Value)
		}
	}
}

// CollectionFormatString returns a Walker.String callback for the
// `collectionFormat:` keyword. Tries the Walker-supplied typed
// string first and falls back to strings.TrimSpace(pr.Value) when
// the grammar's closed-vocab string-enum rejected the source, so
// values outside the OAS v2 vocabulary round-trip verbatim.
//
// SimpleSchema-only — schema-level Validations don't expose
// SetCollectionFormat.
//
// # Details
//
// See [§collection-format-fallback](./README.md#collection-format-fallback)
// for the rationale behind the lax fallback.
func CollectionFormatString(v ifaces.OperationValidationBuilder) func(grammar.Property, string) {
	return func(pr grammar.Property, val string) {
		if pr.Keyword.Name != grammar.KwCollectionFormat {
			return
		}
		x := val
		if x == "" {
			x = strings.TrimSpace(pr.Value)
		}
		if x != "" {
			v.SetCollectionFormat(x)
		}
	}
}

// ComposeString returns a Walker.String callback that fans the
// payload out to every non-nil handler in order. The canonical use
// is to combine PatternString + CollectionFormatString in one
// Walker.String slot.
func ComposeString(hs ...func(grammar.Property, string)) func(grammar.Property, string) {
	return func(pr grammar.Property, val string) {
		for _, h := range hs {
			if h != nil {
				h(pr, val)
			}
		}
	}
}

// Raw returns a Walker.Raw callback for `default:` / `example:` /
// `enum:` (Shape=ShapeRawValue per the lexer table). `default` and
// `example` coerce against scheme via validations.CoerceValue;
// `enum` is delegated to v.SetEnum which routes through
// validations.CoerceEnum inside the adapter.
//
// errSink controls coercion-error semantics:
//
//   - errSink == nil  → swallow silently. The response-header path
//     uses this posture so that a malformed default/example on a
//     header doesn't fail the build.
//   - errSink != nil  → invoked with the first ParseValueFromSchema
//     error. Returning true short-circuits subsequent Raw
//     callbacks within this Walker (the closure's `stopped` flag);
//     returning false continues. Parameters use this to bubble the
//     error up so the build surfaces a malformed default/example
//     as a hard failure.
//
// # Details
//
// See [§raw-errsink](./README.md#raw-errsink) for the per-dispatcher
// wiring and the integration tests that exercise the parameter-path
// hard-failure behaviour.
func Raw(v ifaces.ValidationBuilder, scheme *oaispec.SimpleSchema, errSink func(error) bool) func(grammar.Property) {
	stopped := false
	return func(pr grammar.Property) {
		if stopped {
			return
		}
		switch pr.Keyword.Name {
		case grammar.KwDefault:
			val, err := validations.CoerceValue(pr.Value, scheme)
			if err != nil {
				if errSink != nil && errSink(err) {
					stopped = true
				}
				return
			}
			v.SetDefault(val)
		case grammar.KwExample:
			val, err := validations.CoerceValue(pr.Value, scheme)
			if err != nil {
				if errSink != nil && errSink(err) {
					stopped = true
				}
				return
			}
			v.SetExample(val)
		case grammar.KwEnum:
			v.SetEnum(pr.Value)
		}
	}
}
