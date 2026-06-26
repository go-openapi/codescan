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

// TestCoverage_PatternProperties verifies the patternProperties object schema keyword: each
// annotation line adds a regex → empty-schema entry to schema.patternProperties, and the regex is
// RE2-hygiene-checked like the pattern keyword — an invalid regex is preserved on the schema but
// raises a CodeInvalidAnnotation diagnostic (never dropped silently).
func TestCoverage_PatternProperties(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/pattern-properties/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Valid regexes: both entries land on patternProperties, no diagnostic.
	valid, ok := doc.Definitions["ValidPatterns"]
	require.True(t, ok)
	require.Len(t, valid.PatternProperties, 2)
	assert.Contains(t, valid.PatternProperties, "^[a-z]+$")
	assert.Contains(t, valid.PatternProperties, "^x-")

	// Invalid regex: value preserved (not dropped) ...
	invalid, ok := doc.Definitions["InvalidPattern"]
	require.True(t, ok)
	require.Len(t, invalid.PatternProperties, 1)
	assert.Contains(t, invalid.PatternProperties, "[unclosed(")

	// ... and exactly one CodeInvalidAnnotation diagnostic is raised, naming the patternProperties
	// keyword (mirroring pattern's wording).
	var re2 []grammar.Diagnostic
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidAnnotation {
			re2 = append(re2, d)
		}
	}
	require.Len(t, re2, 1)
	assert.Equal(t, grammar.SeverityWarning, re2[0].Severity)
	assert.Contains(t, re2[0].Message, "patternProperties:")
	assert.Contains(t, re2[0].Message, "not a valid Go RE2 regex")

	scantest.CompareOrDumpJSON(t, doc, "pattern_properties.json")
}
