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

// TestCoverage_Bug3138 verifies the fix for go-swagger issue #3138 ("How To
// mark a field as deprecated?"). OAS2 spec.Schema has no native `deprecated`
// field, so model- and field-level deprecation is emitted as the
// `x-deprecated: true` vendor extension. Two triggers are supported on both
// models and fields, unified by grammar.Block.IsDeprecated():
//
//   - the explicit `deprecated: true` annotation (consumed as a keyword), and
//   - a godoc-style "Deprecated:" paragraph, which stays in the description
//     (the reason text is preserved) and raises no spurious diagnostic.
//
// Operation-level `deprecated: true` keeps using the native OAS2 field.
func TestCoverage_Bug3138(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3138/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Operation-level deprecated still uses the native OAS2 field.
	pi, ok := doc.Paths.Paths["/widgets"]
	require.True(t, ok)
	require.NotNil(t, pi.Get)
	assert.True(t, pi.Get.Deprecated, "operation-level deprecated must be emitted")

	widget := doc.Definitions["Widget"]

	// Field, explicit `deprecated: true`: x-deprecated set, annotation
	// consumed (not left in the description).
	oldName := widget.Properties["oldName"]
	assert.Equal(t, true, oldName.Extensions["x-deprecated"])
	assert.Equal(t, "The legacy name.", oldName.Description)

	// Field, godoc "Deprecated:" paragraph: x-deprecated set AND the reason
	// text is preserved in the description (no spurious diagnostic).
	legacyID := widget.Properties["legacyID"]
	assert.Equal(t, true, legacyID.Extensions["x-deprecated"])
	assert.Contains(t, legacyID.Description, "Deprecated: use Name instead.",
		"the godoc reason must be preserved in the description")

	// A live field is untouched.
	name := widget.Properties["name"]
	assert.NotContains(t, name.Extensions, "x-deprecated")

	// Model, explicit `deprecated: true`.
	assert.Equal(t, true, doc.Definitions["OldWidget"].Extensions["x-deprecated"])

	// Model, godoc "Deprecated:" paragraph.
	retired := doc.Definitions["RetiredWidget"]
	assert.Equal(t, true, retired.Extensions["x-deprecated"])
	assert.Contains(t, retired.Description, "Deprecated: use Widget instead.")

	// The godoc form must NOT be mistaken for a malformed bool keyword.
	for _, d := range diags {
		assert.NotEqual(t, grammar.CodeInvalidBoolean, d.Code,
			"godoc Deprecated paragraph must not raise an invalid-boolean diagnostic: %s", d)
	}

	scantest.CompareOrDumpJSON(t, doc, "bugs_3138_schema.json")
}
