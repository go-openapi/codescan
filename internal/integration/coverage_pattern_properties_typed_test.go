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

// TestCoverage_PatternProperties_Typed locks the typed swagger:patternProperties
// marker (Phase 2c): quoted-regex → typed value (primitive / $ref) pairs,
// RE2-hygiene checking, and the lowest-priority precedence rule.
func TestCoverage_PatternProperties_Typed(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/pattern-properties-typed/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Three typed pairs alongside the named property; the $ref value is discovered.
	typed := doc.Definitions["TypedPatterns"]
	require.Len(t, typed.PatternProperties, 3)
	assert.True(t, typed.PatternProperties["^x-"].Type.Contains("string"))
	assert.True(t, typed.PatternProperties[`^\d+$`].Type.Contains("integer"))
	itemPat := typed.PatternProperties["^item-"]
	assert.Equal(t, "#/definitions/Item", itemPat.Ref.String())
	assert.Contains(t, typed.Properties, "known")
	_, hasItem := doc.Definitions["Item"]
	assert.True(t, hasItem, "referenced pattern value model is discovered")

	// Invalid regex: preserved on the schema, but a CodeInvalidAnnotation warning.
	invalid := doc.Definitions["InvalidTyped"]
	require.Len(t, invalid.PatternProperties, 1)
	assert.Contains(t, invalid.PatternProperties, "[unclosed(")

	// Contradiction: swagger:type wins (string), patternProperties dropped.
	contra := doc.Definitions["Contradiction"]
	assert.True(t, contra.Type.Contains("string"))
	assert.Empty(t, contra.PatternProperties)

	var re2, shape int
	for _, d := range diags {
		switch d.Code {
		case grammar.CodeInvalidAnnotation:
			if assert.Contains(t, d.Message, "patternProperties:") {
				assert.Contains(t, d.Message, "not a valid Go RE2 regex")
			}
			re2++
		case grammar.CodeShapeMismatch:
			shape++
		default:
			// other diagnostics are not asserted here
		}
	}
	assert.Equal(t, 1, re2, "one invalid-regex diagnostic")
	assert.Equal(t, 1, shape, "one precedence diagnostic")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_pattern_properties_typed.json")
}
