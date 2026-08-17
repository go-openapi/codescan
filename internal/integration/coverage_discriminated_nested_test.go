// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

const discriminatedNestedPkg = "./enhancements/discriminated-subtypes-nested/..."

// runNested scans the multi-level-hierarchy fixture, capturing diagnostics by code.
func runNested(t *testing.T, scanModels, prune bool) (*oaispec.Swagger, map[grammar.Code][]grammar.Diagnostic) {
	t.Helper()
	byCode := map[grammar.Code][]grammar.Diagnostic{}
	doc, err := runScan(&codescan.Options{
		Packages:          []string{discriminatedNestedPkg},
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

// assertNestedHierarchy asserts the two-level shape: the root carries its discriminator at the top
// level, the mid-level type is BOTH a subtype (allOf[$ref root, own]) and a base (its own
// discriminator, inside its compound member), and the second-level leaves refer to the mid-level.
func assertNestedHierarchy(t *testing.T, doc *oaispec.Swagger) {
	t.Helper()

	root, ok := doc.Definitions["Shape"]
	require.True(t, ok)
	assert.Equal(t, "shapeType", root.Discriminator, "a plain base carries its discriminator at the top")

	mid, ok := doc.Definitions["Polygon"]
	require.True(t, ok, "the mid-level base must be emitted")
	require.Len(t, mid.AllOf, 2, "the mid-level type is itself a subtype: allOf[root, own]")
	assert.Equal(t, "#/definitions/Shape", mid.AllOf[0].Ref.String())
	assert.Empty(t, mid.Discriminator,
		"a mid-level base has no top-level discriminator — it lives in its compound member")
	assert.Equal(t, "polygonType", mid.AllOf[1].Discriminator,
		"the mid-level type declares its own discriminator inside its own allOf member")

	assert.Equal(t, "#/definitions/Shape", doc.Definitions["Circle"].AllOf[0].Ref.String(),
		"a first-level leaf refers to the root")
	for _, name := range []string{"Square", "Triangle"} {
		leaf, ok := doc.Definitions[name]
		require.Truef(t, ok, "second-level leaf %s must be emitted", name)
		require.Lenf(t, leaf.AllOf, 2, "%s is allOf[mid-level, own]", name)
		assert.Equal(t, "#/definitions/Polygon", leaf.AllOf[0].Ref.String(),
			"%s refers to the mid-level base, not the root", name)
	}
}

// TestDiscriminatedNested_NoScanModels is the cascade case: a route references only the root of a
// two-level hierarchy, and the whole family follows without ScanModels.
//
// It takes two rounds of the reverse index to get there — Polygon is pulled because Shape is
// discriminated, then Square / Triangle are pulled because Polygon (only just pulled in itself) turns
// out to be discriminated too — plus ordinary $ref discovery for Coords, two levels down.
func TestDiscriminatedNested_NoScanModels(t *testing.T) {
	doc, byCode := runNested(t, false, false)

	assertNestedHierarchy(t, doc)

	assert.Contains(t, doc.Definitions, "Coords",
		"an ordinary $ref from a second-level leaf still discovers")
	assert.NotContains(t, doc.Definitions, "Unrelated",
		"the cascade pulls the family, not the package")
	assert.Len(t, doc.Definitions, 6)

	// One Hint per pulled subtype: Polygon + Circle (from Shape), Square + Triangle (from Polygon).
	// Coords arrives through a $ref, so it is not reported as a subtype.
	hints := byCode[grammar.CodeDiscoveredSubtype]
	require.Len(t, hints, 4, "one Hint per subtype pulled, at both levels")
	perBase := map[string]int{}
	for _, d := range hints {
		assert.Equal(t, grammar.SeverityHint, d.Severity)
		assert.Positive(t, d.Pos.Line, "the Hint is located at the subtype's declaration")
		for _, base := range []string{"Shape", "Polygon"} {
			if strings.HasSuffix(d.Message, `discriminated base "`+base+`"`) {
				perBase[base]++
			}
		}
	}
	assert.Equal(t, map[string]int{"Shape": 2, "Polygon": 2}, perBase,
		"two subtypes reported per level, each naming the base that pulled it")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_discriminated_nested.json")
}

// TestDiscriminatedNested_ScanModels is the control: ScanModels emits the hierarchy plus the
// unreferenced model, and the reverse index reports nothing (everything is built up front).
func TestDiscriminatedNested_ScanModels(t *testing.T) {
	doc, byCode := runNested(t, true, false)

	assertNestedHierarchy(t, doc)

	assert.Contains(t, doc.Definitions, "Unrelated")
	assert.Len(t, doc.Definitions, 7)
	assert.Empty(t, byCode[grammar.CodeDiscoveredSubtype])

	scantest.CompareOrDumpJSON(t, doc, "enhancements_discriminated_nested_all.json")
}

// TestDiscriminatedNested_Pruned locks the prune half on a multi-level hierarchy: the reachability
// rule has to fire on the mid-level base too — whose discriminator sits in its compound member — or
// the second level would be pruned away under ScanModels + PruneUnusedModels.
func TestDiscriminatedNested_Pruned(t *testing.T) {
	doc, byCode := runNested(t, true, true)

	assertNestedHierarchy(t, doc)

	assert.Contains(t, doc.Definitions, "Coords")
	assert.NotContains(t, doc.Definitions, "Unrelated")
	assert.Len(t, doc.Definitions, 6)

	require.Len(t, byCode[grammar.CodePrunedUnused], 1, "only the unreferenced model is pruned")
}
