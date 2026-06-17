// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestKeywordCrossPackageLeaf locks cross-package leaf resolution for the
// type-name keyword sites. A bare type name that is NOT in the annotating
// type's own package resolves by leaf against the discovered model set:
// uniquely -> the model's definition; ambiguously -> diagnostic + drop. This
// matches the leaf resolution the name-identity engine gave routes/responses.
func TestKeywordCrossPackageLeaf(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./enhancements/keyword-xpkg-leaf/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// swagger:additionalProperties Widget (cross-pkg, unique) -> $ref.
	bag := doc.Definitions["Bag"]
	require.NotNil(t, bag.AdditionalProperties)
	require.NotNil(t, bag.AdditionalProperties.Schema)
	assert.Equal(t, "#/definitions/Widget", bag.AdditionalProperties.Schema.Ref.String())

	// swagger:patternProperties "^item-": Widget (cross-pkg, unique) -> $ref.
	cat := doc.Definitions["Catalog"]
	require.Contains(t, cat.PatternProperties, "^item-")
	itemPat := cat.PatternProperties["^item-"]
	assert.Equal(t, "#/definitions/Widget", itemPat.Ref.String())

	// swagger:type Gadget (cross-pkg, unique) -> inlined structure (no $ref).
	holder := doc.Definitions["Holder"]
	thing := holder.Properties["thing"]
	assert.Empty(t, thing.Ref.String(), "swagger:type inlines; no $ref")
	assert.Contains(t, thing.Properties, "power", "Gadget structure inlined in place")

	// swagger:additionalProperties Thing (cross-pkg, AMBIGUOUS) -> dropped.
	amb := doc.Definitions["AmbiguousBag"]
	assert.Nil(t, amb.AdditionalProperties, "ambiguous leaf is dropped, not guessed")
	assert.True(t, hasDiagnostic(diags, grammar.CodeAmbiguousTypeName), "ambiguity is diagnosed")

	// the referenced models are present (Widget/Gadget unique -> bare).
	assert.Contains(t, doc.Definitions, "Widget")
	assert.Contains(t, doc.Definitions, "Gadget")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_keyword_xpkg_leaf.json")
}
