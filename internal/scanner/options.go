// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package scanner

import (
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/spec"
)

// Options configures a scan. The zero value is a valid configuration:
// every flag defaults to false and every slice/map defaults to nil.
//
// # Details
//
// See [§options](./README.md#options) for the field overview, and
// [§descwithref](./README.md#descwithref) and
// [§diagnostics](./README.md#diagnostics) for the two fields with
// non-trivial semantics (DescWithRef and OnDiagnostic).
type Options struct {
	Packages                []string
	InputSpec               *spec.Swagger
	ScanModels              bool
	WorkDir                 string
	BuildTags               string
	ExcludeDeps             bool
	Include                 []string
	Exclude                 []string
	IncludeTags             []string
	ExcludeTags             []string
	SetXNullableForPointers bool
	RefAliases              bool // aliases result in $ref, otherwise aliases are expanded
	TransparentAliases      bool // aliases are completely transparent, never creating definitions
	// DescWithRef controls description preservation on $ref'd fields
	// in the description-only-decoration case: when a struct field's
	// Go type resolves to a named type ($ref) and its only
	// field-level decoration is a description (no validations, no
	// user-authored extensions).
	//
	//   - false (default): the description is dropped and the field
	//     emits as a bare `{$ref: ...}`.
	//   - true: the description is preserved by wrapping the $ref in
	//     a single-arm `allOf` compound — `{description: "...",
	//     allOf: [{$ref}]}` — the JSON-Schema-draft-4 correct shape
	//     for sibling description.
	//
	// When the field also carries validation overrides (pattern,
	// enum, example, etc.) or user-authored vendor extensions, the
	// allOf compound is mandatory regardless of this flag — the
	// override would be lost otherwise.
	//
	// See [§descwithref](./README.md#descwithref).
	DescWithRef    bool
	SkipExtensions bool // skip generating x-go-* vendor extensions in the spec

	// SkipEnumDescriptions controls whether the per-enum-value const-name
	// mapping built from `swagger:enum` (e.g. "FIRST TestEnumFirst") is
	// folded into the property / parameter / header `description`.
	//
	//   - false (default): the mapping is appended to the authored
	//     description AND exposed via the `x-go-enum-desc` vendor extension
	//     (backward-compatible behaviour).
	//   - true: the description is left as the authored prose; the mapping
	//     rides `x-go-enum-desc` only.
	//
	// Independent of SkipExtensions: with SkipExtensions also set, the
	// mapping is suppressed everywhere. See go-swagger/go-swagger#2922.
	SkipEnumDescriptions bool

	Debug bool // enable verbose debug logging during scanning

	// OnDiagnostic, when non-nil, is invoked for every diagnostic the
	// builder layer records (lexer/parser warnings, semantic-validation
	// failures from the validations package, etc.). The callback fires
	// once per diagnostic in source order; diagnostics never block the
	// build — invalid constructs are silently dropped from the output
	// spec while their explanation flows through this channel.
	//
	// Experimental: the public API surface for diagnostics is subject
	// to change while LSP integration matures. See
	// [§diagnostics](./README.md#diagnostics).
	OnDiagnostic func(grammar.Diagnostic)
}
