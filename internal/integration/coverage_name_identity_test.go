// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// Name-identity / cyclic-$ref corpus.
//
//   - CONTROLS (recursion, no-collision) are already correct and DETERMINISTIC;
//     their goldens MUST stay byte-identical through every stage (G3 legitimate
//     recursion, G5 zero-churn common case).
//   - COLLISION fixtures (3-way, mixed, same-package-dup, deep) used to MERGE
//     distinct types non-deterministically. They now stay distinct and
//     deterministic: cross-package collisions are qualified by a minimal-depth
//     PascalCase concat of their nearest package segments (b.Test -> BTest),
//     with a diagnostic; same-package duplicates revert the loser to its Go name.
//
// Triager-flagged name-conflict family covered here (the whole family resolves at once via this one
// engine; the merge should `contributes` all of them):
//
//   - #2783 — models mixed across packages       (TestNameIdentity_3Way/Mixed, Bug2783)
//   - #2637 — self-cyclic $ref, same-named type   (Bug2637)
//   - #2662 — same ref names -> invalid spec       (≡ #2783 mechanism)
//   - #1734 — same name service/deps -> nondeterm. (≡ determinism, G4)
//   - #2126 — same-named models, examples not mixed (TestNameIdentity_ExamplesNoMix)
//   - #2398 — warn on duplicate definitions        (CodeDuplicateModelName / CodeCollidingModelName asserts)

func nameIdentityDoc(t *testing.T, pkg string) *oaispec.Swagger {
	t.Helper()
	doc, _ := nameIdentityDocDiags(t, pkg)
	return doc
}

func nameIdentityDocDiags(t *testing.T, pkg string) (*oaispec.Swagger, []grammar.Diagnostic) {
	t.Helper()
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{pkg},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc, diags
}

// hasDiagnostic reports whether any captured diagnostic has the given code.
func hasDiagnostic(diags []grammar.Diagnostic, code grammar.Code) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// --- CONTROLS (deterministic; golden-locked) ----------------------------------

// TestNameIdentity_Recursion is the G3 control: legitimate recursion (direct self-reference
// Node.next, and mutual Loop<->Knot) produces property back-$refs to ancestor definitions, which is
// valid OAS and must keep working unchanged through every engine stage.
func TestNameIdentity_Recursion(t *testing.T) {
	doc := nameIdentityDoc(t, "./enhancements/name-identity-recursion/...")

	require.Len(t, doc.Definitions, 3, "Node, Loop, Knot are three distinct definitions")
	require.Contains(t, doc.Definitions, "Node")
	require.Contains(t, doc.Definitions, "Loop")
	require.Contains(t, doc.Definitions, "Knot")

	// G3: a property back-$ref to the ancestor definition is valid and present.
	node := doc.Definitions["Node"]
	next := node.Properties["next"]
	assert.Equal(t, "#/definitions/Node", next.Ref.String(),
		"direct self-recursion is a valid property $ref, not a self-ref body")
	assert.Empty(t, node.Ref.String(),
		"the Node definition body itself must NOT be a $ref")
	loop := doc.Definitions["Loop"]
	knot := doc.Definitions["Knot"]
	knotRef := loop.Properties["knot"]
	loopRef := knot.Properties["loop"]
	assert.Equal(t, "#/definitions/Knot", knotRef.Ref.String())
	assert.Equal(t, "#/definitions/Loop", loopRef.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_recursion.json")
}

// TestNameIdentity_NoCollision is the G5 control: distinct names across packages, no collision.
//
// Its golden must stay byte-identical through every stage — proof the engine introduces zero
// churn when there is nothing to deconflict.
func TestNameIdentity_NoCollision(t *testing.T) {
	doc := nameIdentityDoc(t, "./enhancements/name-identity-no-collision/...")

	require.Len(t, doc.Definitions, 2, "Alpha + Beta, distinct")
	alpha := doc.Definitions["Alpha"]
	beta := alpha.Properties["beta"]
	assert.Equal(t, "#/definitions/Beta", beta.Ref.String(),
		"bare-name $ref in the no-collision common case")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_no_collision.json")
}

// --- COLLISION fixtures (distinct + deterministic; concat-qualified) -----------

// TestNameIdentity_3Way: three packages each declare `swagger:model Widget`.
//
// Distinct, one-segment concat: AWidget / BWidget / CWidget.
func TestNameIdentity_3Way(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./enhancements/name-identity-3way/...")

	assert.Len(t, doc.Definitions, 4, "Container + three distinct Widgets")
	for _, name := range []string{"Container", "AWidget", "BWidget", "CWidget"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.NotContains(t, doc.Definitions, "Widget", "no merged bare Widget key")
	assert.True(t, hasDiagnostic(diags, grammar.CodeCollidingModelName), "collision is diagnosed")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_3way.json")
}

// TestNameIdentity_Mixed: BOTH collision kinds resolve distinctly — explicit (x.Item / y.Item,
// each `swagger:model Item`) and implicit (x.Record / y.Record, unannotated, discovered via
// reference).
func TestNameIdentity_Mixed(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./enhancements/name-identity-mixed/...")

	assert.Len(t, doc.Definitions, 5, "Root + two Items + two Records")
	for _, name := range []string{"Root", "XItem", "YItem", "XRecord", "YRecord"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.True(t, hasDiagnostic(diags, grammar.CodeCollidingModelName))

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_mixed.json")
}

// TestNameIdentity_SamePkgDup (D-4): two DIFFERENT types in the SAME package both claim
// `swagger:model Dup`.
//
// The first keeps "Dup"; the duplicate reverts to its Go name ("Second") with a
// same-package-duplicate diagnostic. (No cross-package collision here, so the names stay bare.)
func TestNameIdentity_SamePkgDup(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./enhancements/name-identity-same-pkg-dup/...")

	assert.Len(t, doc.Definitions, 3, "Dup (First) + Second (fallback) + Root")
	assert.Contains(t, doc.Definitions, "Dup", "first declaration keeps the overridden name")
	assert.Contains(t, doc.Definitions, "Second", "the duplicate reverts to its Go name")
	assert.Contains(t, doc.Definitions, "Root")
	assert.True(t, hasDiagnostic(diags, grammar.CodeDuplicateModelName), "duplicate is diagnosed")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_same_pkg_dup.json")
}

// TestNameIdentity_Deep: a/mongo.Book and b/mongo.Book share both the leaf name AND the one-segment
// concat ("MongoBook"), so the concat rung must deepen to two segments: AMongoBook / BMongoBook.
func TestNameIdentity_Deep(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./enhancements/name-identity-deep/...")

	assert.Len(t, doc.Definitions, 3, "Library + two distinct Books")
	for _, name := range []string{"Library", "AMongoBook", "BMongoBook"} {
		assert.Contains(t, doc.Definitions, name)
	}
	assert.NotContains(t, doc.Definitions, "MongoBook", "one-segment concat collides; must deepen")
	assert.True(t, hasDiagnostic(diags, grammar.CodeCollidingModelName))

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_deep.json")
}

// TestNameIdentity_RecursionCollision is the cyclic-within-collision guard: p.Node and q.Node are
// each self-recursive AND collide on "Node".
//
// They must resolve to distinct names (PNode / QNode), and — the crucial part — each
// self-`$ref` must be rewritten in lockstep with its OWN renamed key (a ref that points into the
// very group being renamed), never dangling at the pre-reduce deep key and never pointing at the
// sibling.
func TestNameIdentity_RecursionCollision(t *testing.T) {
	doc, diags := nameIdentityDocDiags(t, "./enhancements/name-identity-recursion-collision/...")

	require.Len(t, doc.Definitions, 3, "Tree + PNode + QNode")
	for _, name := range []string{"Tree", "PNode", "QNode"} {
		require.Contains(t, doc.Definitions, name)
	}
	assert.NotContains(t, doc.Definitions, "Node", "no merged bare Node key")
	assert.True(t, hasDiagnostic(diags, grammar.CodeCollidingModelName))

	// Each self-ref points at its OWN renamed definition (lockstep rewrite), and the body is not a
	// degenerate self-`$ref` (G2/G3).
	pNode := doc.Definitions["PNode"]
	qNode := doc.Definitions["QNode"]
	assert.Empty(t, pNode.Ref.String(), "PNode body must not be a self-$ref")
	pNext := pNode.Properties["next"]
	qNext := qNode.Properties["next"]
	assert.Equal(t, "#/definitions/PNode", pNext.Ref.String(), "p.Node.next -> PNode")
	assert.Equal(t, "#/definitions/QNode", qNext.Ref.String(), "q.Node.next -> QNode")

	// The container's cross-refs resolve to the qualified names too.
	tree := doc.Definitions["Tree"]
	tp := tree.Properties["p"]
	tq := tree.Properties["q"]
	assert.Equal(t, "#/definitions/PNode", tp.Ref.String())
	assert.Equal(t, "#/definitions/QNode", tq.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_recursion_collision.json")
}

// TestNameIdentity_Hierarchical exercises the hierarchical fail-safe: two long-named packages each
// declare `swagger:model Config`, whose flat minimal concat exceeds the readability budget.
//
// By default the (always-correct) flat concat is kept; with EmitHierarchicalNames the over-budget
// group is emitted as nested container definitions instead.
func TestNameIdentity_Hierarchical(t *testing.T) {
	const pkg = "./enhancements/name-identity-hierarchical/..."

	t.Run("flat concat by default", func(t *testing.T) {
		doc := nameIdentityDoc(t, pkg)

		require.Len(t, doc.Definitions, 3, "Settings + two flat concats")
		for _, name := range []string{"Settings", "RecommendationengineConfig", "NotificationserviceConfig"} {
			assert.Contains(t, doc.Definitions, name)
		}
		assert.NotContains(t, doc.Definitions, "recommendationengine", "no container node in flat mode")

		settings := doc.Definitions["Settings"]
		rec := settings.Properties["rec"]
		assert.Equal(t, "#/definitions/RecommendationengineConfig", rec.Ref.String())

		scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_hierarchical_flat.json")
	})

	t.Run("nested when EmitHierarchicalNames is set", func(t *testing.T) {
		var diags []grammar.Diagnostic
		doc, err := codescan.Run(&codescan.Options{
			Packages:              []string{pkg},
			WorkDir:               scantest.FixturesDir(),
			ScanModels:            true,
			EmitHierarchicalNames: true,
			OnDiagnostic:          func(d grammar.Diagnostic) { diags = append(diags, d) },
		})
		require.NoError(t, err)
		require.NotNil(t, doc)

		require.Len(t, doc.Definitions, 3, "Settings + two container nodes")
		for _, name := range []string{"Settings", "recommendationengine", "notificationservice"} {
			assert.Contains(t, doc.Definitions, name)
		}
		assert.NotContains(t, doc.Definitions, "RecommendationengineConfig", "flat concat replaced by nesting")

		// the container is a lenient node carrying the nested model.
		container := doc.Definitions["recommendationengine"]
		require.NotNil(t, container.AdditionalProperties)
		assert.True(t, container.AdditionalProperties.Allows, "container is additionalProperties:true")
		nested, ok := container.ExtraProps["Config"].(oaispec.Schema)
		require.True(t, ok, "nested Config model lives under the container")
		assert.Contains(t, nested.Properties, "enabled")

		// refs resolve to the nested pointer.
		settings := doc.Definitions["Settings"]
		rec := settings.Properties["rec"]
		assert.Equal(t, "#/definitions/recommendationengine/Config", rec.Ref.String())

		assert.True(t, hasDiagnostic(diags, grammar.CodeHierarchicalModelName), "hierarchical emission is diagnosed")

		scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_hierarchical_nested.json")
	})
}

// TestNameIdentity_ExamplesNoMix locks go-swagger #2126: two packages each declare `swagger:model
// Widget` with the SAME json field but DIFFERENT example values.
//
// The fix keeps the definitions distinct (AWidget / BWidget), so each retains its OWN example —
// neither package's example bleeds into the other's (the merge used to clobber one with the other).
func TestNameIdentity_ExamplesNoMix(t *testing.T) {
	doc := nameIdentityDoc(t, "./enhancements/name-identity-examples-no-mix/...")

	require.Len(t, doc.Definitions, 3, "Catalog + AWidget + BWidget")
	aw, okA := doc.Definitions["AWidget"]
	bw, okB := doc.Definitions["BWidget"]
	require.True(t, okA)
	require.True(t, okB)
	assert.Equal(t, "from-a", aw.Properties["label"].Example, "AWidget keeps a's example")
	assert.Equal(t, "from-b", bw.Properties["label"].Example, "BWidget keeps b's example")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_name_identity_examples_no_mix.json")
}
