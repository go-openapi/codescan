// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

// Package diagnostics carries fixtures that exercise the schema
// builder's diagnostic emission path. Each type is annotated with
// at least one deliberately invalid construct that the grammar
// parser rejects with a non-fatal diagnostic; the offending value
// is dropped from the output spec while the diagnostic flows
// through Builder.Diagnostics() / Options.OnDiagnostic.
package diagnostics

// BadMaximum has an invalid maximum: value. The parser emits a
// CodeInvalidNumber diagnostic and the maximum keyword is dropped
// from the output schema.
//
// swagger:model BadMaximum
type BadMaximum struct {
	// Count holds an arbitrary count.
	//
	// maximum: notanumber
	Count int `json:"count"`
}

// Helper types for the AmbiguousEmbed fixture: two unrelated structs
// that each carry a property whose JSON name is "shared", but whose
// Go field names differ. Embedding both into a single parent triggers
// the same-depth ambiguity case Go itself would refuse to promote.

// SharedFoo carries a `shared` JSON property under Go name Foo.
//
// swagger:model SharedFoo
type SharedFoo struct {
	Foo string `json:"shared"`
}

// SharedBar carries a `shared` JSON property under Go name Bar.
//
// swagger:model SharedBar
type SharedBar struct {
	Bar string `json:"shared"`
}

// AmbiguousEmbed embeds two unrelated types that both promote the
// `shared` JSON property under different Go names. The schema
// builder emits a CodeAmbiguousEmbed diagnostic; the resulting
// spec is last-write-wins (Bar overrides Foo because struct fields
// iterate in source order — SharedBar is the later embed).
//
// swagger:model AmbiguousEmbed
type AmbiguousEmbed struct {
	SharedFoo
	SharedBar
}

// Helper types for the DepthShadowingEmbed fixture: an inner struct
// shadowed by an outer struct's own explicit field of the same JSON
// name. Go's depth rule prefers the shallower field; codescan's
// last-write-wins produces the same outcome (the explicit field
// processes after the embed pass). The diagnostic must NOT fire.

// DepthInner carries `shared` under Go name Foo at the deeper layer.
//
// swagger:model DepthInner
type DepthInner struct {
	Foo string `json:"shared"`
}

// DepthMiddle re-declares `shared` under Go name Bar at its own
// (shallower) depth, on top of an embed of DepthInner.
//
// swagger:model DepthMiddle
type DepthMiddle struct {
	DepthInner
	Bar string `json:"shared"`
}

// DepthShadowingEmbed exercises legitimate Go-depth-rule shadowing
// across an embed chain: DepthMiddle.Bar (depth 1 from the parent
// after embedding) shadows DepthInner.Foo (depth 2). The diagnostic
// must remain silent.
//
// swagger:model DepthShadowingEmbed
type DepthShadowingEmbed struct {
	DepthMiddle
}

// ExplicitOverride exercises the explicit-override case: a top-level
// struct embeds a type carrying `shared`, then re-declares `shared`
// as its own field. The explicit field wins at embedDepth = 0; the
// diagnostic must remain silent.
//
// swagger:model ExplicitOverride
type ExplicitOverride struct {
	SharedFoo
	Bar string `json:"shared"`
}
