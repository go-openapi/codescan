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

const discriminatedSubtypesPkg = "./enhancements/discriminated-subtypes/..."

// runSubtypes scans the discriminated-subtypes fixture, capturing diagnostics by code.
func runSubtypes(t *testing.T, scanModels, prune bool) (*oaispec.Swagger, map[grammar.Code][]grammar.Diagnostic) {
	t.Helper()
	byCode := map[grammar.Code][]grammar.Diagnostic{}
	doc, err := runScan(&codescan.Options{
		Packages:          []string{discriminatedSubtypesPkg},
		WorkDir:           scantest.FixturesDir(),
		ScanModels:        scanModels,
		PruneUnusedModels: prune,
		OnDiagnostic: func(d grammar.Diagnostic) {
			byCode[d.Code] = append(byCode[d.Code], d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	return doc, byCode
}

// assertPolymorphicFamily asserts the shape the subtypes must keep however they were discovered:
// the base carries the discriminator and every subtype is `allOf: [{$ref base}, {own props}]`.
func assertPolymorphicFamily(t *testing.T, doc *oaispec.Swagger) {
	t.Helper()

	base, ok := doc.Definitions["TeslaCar"]
	require.True(t, ok, "the referenced base is always emitted")
	assert.Equal(t, "model", base.Discriminator, "the base carries the discriminator")

	for _, name := range []string{"modelS", "modelX", "modelY"} {
		sub, ok := doc.Definitions[name]
		require.Truef(t, ok, "subtype %s must be emitted", name)
		require.Lenf(t, sub.AllOf, 2, "subtype %s is allOf[base, own]", name)
		assert.Equal(t, "#/definitions/TeslaCar", sub.AllOf[0].Ref.String(),
			"subtype %s refers to the base", name)
	}
}

// TestDiscriminatedSubtypes_NoScanModels is the core case (go-swagger#1913): a route references only
// the discriminated base, and its subtypes come along WITHOUT ScanModels — pulled by the reverse
// `swagger:allOf` index, since nothing in the document $refs them.
//
// Battery proves a pulled subtype keeps discovering (it is reached only from modelS), and the two
// negative controls prove the pull is not a blanket "emit every model": Unrelated is unreferenced,
// and PlainSub's base carries no discriminator.
func TestDiscriminatedSubtypes_NoScanModels(t *testing.T) {
	doc, byCode := runSubtypes(t, false, false)

	assertPolymorphicFamily(t, doc)

	assert.Contains(t, doc.Definitions, "Battery",
		"a pulled subtype keeps discovering its own dependencies")
	battery := doc.Definitions["modelS"].AllOf[1].Properties["battery"]
	assert.Equal(t, "#/definitions/Battery", battery.Ref.String())

	for _, name := range []string{"Unrelated", "PlainBase", "PlainSub"} {
		assert.NotContainsf(t, doc.Definitions, name,
			"%s is not part of a discriminated family and must not be pulled in", name)
	}
	// The edges family is discriminated too, but its base never enters the reachable set — the pull is
	// gated on the base being emitted, not on it merely existing in the source.
	for _, name := range []string{"Vehicle", "Bike", "Trike", "Hidden", "Plain"} {
		assert.NotContainsf(t, doc.Definitions, name,
			"%s belongs to a family no route reaches and must stay out", name)
	}
	assert.Len(t, doc.Definitions, 5)

	// One located Hint per pulled subtype (modelS, modelX, modelY) — Battery arrives through an
	// ordinary $ref, not the reverse index, so it is not reported as a subtype.
	hints := byCode[grammar.CodeDiscoveredSubtype]
	require.Len(t, hints, 3, "one Hint per subtype pulled by the reverse index")
	for _, d := range hints {
		assert.Equal(t, grammar.SeverityHint, d.Severity, "a subtype pull is informational")
		assert.Positive(t, d.Pos.Line, "the Hint is located at the subtype's declaration")
	}

	scantest.CompareOrDumpJSON(t, doc, "enhancements_discriminated_subtypes.json")
}

// TestDiscriminatedSubtypes_ScanModels is the control: under ScanModels every swagger:model is
// emitted anyway — including the two negative controls — so the reverse index changes nothing and
// reports no pull.
func TestDiscriminatedSubtypes_ScanModels(t *testing.T) {
	doc, byCode := runSubtypes(t, true, false)

	assertPolymorphicFamily(t, doc)

	for _, name := range []string{
		"Battery", "Unrelated", "PlainBase", "PlainSub",
		"Vehicle", "Bike", "Trike", "Hidden", "Plain",
	} {
		assert.Containsf(t, doc.Definitions, name, "ScanModels emits every model, including %s", name)
	}
	assert.Len(t, doc.Definitions, 13)
	assert.Empty(t, byCode[grammar.CodeDiscoveredSubtype],
		"under ScanModels the subtypes are built up front, so nothing is pulled")

	// The edge shapes: a pointer embed and an alias embed both compound onto the base; an ignored embed
	// and a plain embed do not (the latter inlines the base's properties instead).
	for _, name := range []string{"Bike", "Trike"} {
		sub := doc.Definitions[name]
		require.Lenf(t, sub.AllOf, 2, "%s is allOf[base, own]", name)
		assert.Equal(t, "#/definitions/Vehicle", sub.AllOf[0].Ref.String())
	}
	assert.Empty(t, doc.Definitions["Hidden"].AllOf, "an ignored embed emits no allOf member")
	assert.Empty(t, doc.Definitions["Plain"].AllOf, "a plain embed inlines instead")
	assert.Contains(t, doc.Definitions["Plain"].Properties, "kind",
		"a plain embed promotes the base's properties")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_discriminated_subtypes_all.json")
}

// TestDiscriminatedSubtypes_Pruned is the prune-reachability half (the discriminator rule §12
// deferred to §15): under ScanModels + PruneUnusedModels the subtypes survive, because a reachable
// discriminated base keeps its family — even though no $ref reaches them — while the models outside
// the family are still pruned.
func TestDiscriminatedSubtypes_Pruned(t *testing.T) {
	doc, byCode := runSubtypes(t, true, true)

	assertPolymorphicFamily(t, doc)

	assert.Contains(t, doc.Definitions, "Battery",
		"a subtype's own dependencies stay reachable through it")
	for _, name := range []string{"Unrelated", "PlainBase", "PlainSub"} {
		assert.NotContainsf(t, doc.Definitions, name, "%s is outside the family and is pruned", name)
	}
	// A discriminated base that is itself unreachable does NOT drag its family back in: the rule keeps a
	// REACHED base's family, it does not make bases roots.
	for _, name := range []string{"Vehicle", "Bike", "Trike", "Hidden", "Plain"} {
		assert.NotContainsf(t, doc.Definitions, name,
			"%s hangs off an unreachable base and is pruned with it", name)
	}
	assert.Len(t, doc.Definitions, 5)

	require.Len(t, byCode[grammar.CodePrunedUnused], 8, "one Hint per pruned definition")
}
