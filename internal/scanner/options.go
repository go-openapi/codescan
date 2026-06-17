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

	// NameConcatBudget tunes the readability cutoff used when the
	// name-identity reduce stage deconflicts colliding definition names
	// by concatenating package segments (b.Test / c.Test -> BTest /
	// CTest). Each candidate concat is scored in [0,1] — lower is more
	// readable (shorter overall, fewer parts, no over-long segment). A
	// collision group whose best concat scores ABOVE the budget is a
	// candidate for the hierarchical fallback (name-identity Stage 3 /
	// K3).
	//
	// The zero value selects the built-in default (0.65). Raise it
	// toward 1.0 to accept longer concats; lower it to fall back sooner.
	NameConcatBudget float64

	// EmitHierarchicalNames enables the hierarchical fail-safe for the
	// rare collision groups whose best flat concat exceeds
	// NameConcatBudget. When set, such a group is emitted as nested
	// container definitions (`#/definitions/<pkg>/<Name>`, with
	// `additionalProperties:true` + `x-go-package` on each container)
	// instead of a long flat concat, and an explanatory diagnostic is
	// raised.
	//
	// Default false — and deliberately so: a nested definition is a deep
	// JSON pointer that only `ExpandSpec` resolves, and a definitions-
	// enumerating consumer (e.g. go-swagger codegen, one model per entry)
	// sees the container nodes rather than the models. The always-correct
	// flat concat stays the default; enable this only when you prefer the
	// nested shape for the over-budget tail.
	EmitHierarchicalNames bool

	// EmitXGoType stamps an `x-go-type` vendor extension on every emitted
	// definition, recording the fully-qualified originating Go type
	// (`<package path>.<type name>`) alongside the existing `x-go-name` /
	// `x-go-package` traceability extensions.
	//
	//   - false (default): no `x-go-type` is emitted for ordinary types
	//     (the extension still appears on the narrow special-type cases
	//     that have always carried it — `error`, the unmodellable
	//     generic-type fallback).
	//   - true: each definition carries `x-go-type`, useful for
	//     round-tripping a generated spec back to its source Go types.
	//
	// Under the SkipExtensions umbrella: with SkipExtensions also set,
	// no vendor extension is emitted regardless. See
	// go-swagger/go-swagger#2924.
	EmitXGoType bool

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
