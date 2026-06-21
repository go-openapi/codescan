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
	// Deprecated: prefer EmitRefSiblings, which preserves description
	// AND extensions as direct $ref siblings (the modern, lenient
	// shape). DescWithRef is retained with its original semantics (the
	// strict draft-4 single-arm allOf wrap for the description-only
	// case) and remains a no-op when EmitRefSiblings is set.
	//
	// See [§ref-override](../builders/schema/README.md#ref-override).
	DescWithRef bool

	// EmitRefSiblings emits a $ref'd field's description and vendor
	// extensions as DIRECT siblings of the `$ref`
	// (`{$ref, description, x-*}`) instead of wrapping them in an allOf
	// compound. Strict JSON-Schema-draft-4 ignores siblings of `$ref`
	// (hence the default allOf wrap), but OpenAPI 3.1 / JSON Schema
	// 2020-12 and most modern Swagger-UI renderers honour them.
	//
	//   - false (default): description / extensions follow the legacy
	//     wrap behaviour (extensions lift onto a single-arm allOf;
	//     description-only is governed by DescWithRef).
	//   - true: description and extensions ride directly alongside the
	//     `$ref`, no allOf.
	//
	// Validations and externalDocs are NOT siblings-eligible: when
	// present they still force an allOf compound (validations on the
	// override arm), and description / extensions then ride the outer
	// compound. This flag changes only the no-forced-compound cases.
	//
	// See [§ref-override](../builders/schema/README.md#ref-override).
	EmitRefSiblings bool

	// SkipAllOfCompounding disables the allOf-compound rewrite for
	// $ref'd struct fields entirely: no allOf compound is ever emitted.
	//
	//   - false (default): siblings are preserved via the allOf compound
	//     (or, under EmitRefSiblings, as direct $ref siblings).
	//   - true: no compound is produced. Validations and externalDocs —
	//     which can only ride a compound — are DROPPED. Description and
	//     extensions are likewise dropped UNLESS EmitRefSiblings is also
	//     set, in which case they survive as direct `$ref` siblings.
	//     Every drop raises one diagnostic through OnDiagnostic — the
	//     loss is never silent.
	//
	// `required:` is a parent-side concern (it lands on the enclosing
	// object's `required` list, not as a $ref sibling) and is preserved
	// regardless of this flag.
	//
	// Intended for downstream consumers (e.g. go-swagger codegen) that
	// expect a bare `$ref` for a field pointing at a model and do not
	// handle the allOf-compounded shape. See [§ref-override].
	SkipAllOfCompounding bool

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

	// SingleLineCommentAsDescription routes a single-line doc comment to
	// the object's `description` regardless of trailing punctuation,
	// never to `title` / `summary`.
	//
	//   - false (default): the first-sentence convention applies — a
	//     single-line comment ending in punctuation (`.`, `!`, `?`)
	//     becomes the `title` (model / info) or `summary` (operation);
	//     without trailing punctuation it is a `description`.
	//   - true: a single-line comment is always a `description`. Multi-
	//     line comments keep the existing title/description split (the
	//     first line, or the paragraph before the first blank line, is
	//     still the title).
	//
	// See go-swagger/go-swagger#2626.
	SingleLineCommentAsDescription bool

	// Debug is deprecated and has no effect.
	//
	// It formerly enabled verbose debug logging to stderr during scanning.
	// That logger was retired: scan-time observations now flow exclusively
	// through OnDiagnostic (which the caller routes to a logger of their
	// choice), and codescan no longer writes to stdout/stderr — keeping it
	// usable from a TUI or a WASI/WASM host.
	//
	// Deprecated: wire OnDiagnostic instead. This field is retained for API
	// compatibility and is ignored.
	Debug bool

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

	// OnProvenance, when non-nil, is invoked once per anchor node in the
	// produced spec, carrying its JSON pointer and the source position of
	// the Go construct that produced it (see [Provenance]). Anchors are
	// code-detail nodes (type decls, fields, values, route/meta blocks);
	// finer nodes resolve to their nearest anchored ancestor at the
	// consumer. The callback never blocks the build.
	//
	// Experimental: the cross-ref surface may change while LSP / TUI
	// integration matures.
	OnProvenance func(Provenance)
}
