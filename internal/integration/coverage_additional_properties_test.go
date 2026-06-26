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

// TestCoverage_AdditionalProperties locks the type-level swagger:additionalProperties marker
// (go-swagger#2539 / #3005, forthcoming §17): true / false / typed value / model $ref, the
// map-override case, coexistence with maxProperties, and the lowest-priority precedence rule.
func TestCoverage_AdditionalProperties(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/additional-properties/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// false → SchemaOrBool{Allows:false}, named props kept (complement).
	forbid := doc.Definitions["ForbidExtras"]
	require.NotNil(t, forbid.AdditionalProperties)
	assert.Nil(t, forbid.AdditionalProperties.Schema)
	assert.False(t, forbid.AdditionalProperties.Allows, "additionalProperties: false")
	assert.Contains(t, forbid.Properties, "a")
	assert.Contains(t, forbid.Properties, "b")

	// true → SchemaOrBool{Allows:true}.
	allow := doc.Definitions["AllowExtras"]
	require.NotNil(t, allow.AdditionalProperties)
	assert.Nil(t, allow.AdditionalProperties.Schema)
	assert.True(t, allow.AdditionalProperties.Allows, "additionalProperties: true")

	// typed value (integer), complement.
	typed := doc.Definitions["TypedExtras"]
	require.NotNil(t, typed.AdditionalProperties)
	require.NotNil(t, typed.AdditionalProperties.Schema)
	assert.True(t, typed.AdditionalProperties.Schema.Type.Contains("integer"))
	assert.Contains(t, typed.Properties, "a")

	// model reference → $ref value schema; the referenced model is emitted.
	ref := doc.Definitions["RefExtras"]
	require.NotNil(t, ref.AdditionalProperties)
	require.NotNil(t, ref.AdditionalProperties.Schema)
	assert.Equal(t, "#/definitions/Thing", ref.AdditionalProperties.Schema.Ref.String())
	_, hasThing := doc.Definitions["Thing"]
	assert.True(t, hasThing, "referenced value model is discovered")

	// map element schema overridden (string → integer).
	override := doc.Definitions["OverrideMap"]
	require.NotNil(t, override.AdditionalProperties)
	require.NotNil(t, override.AdditionalProperties.Schema)
	assert.True(t, override.AdditionalProperties.Schema.Type.Contains("integer"),
		"marker overrides the map element schema")

	// coexists with maxProperties.
	bounded := doc.Definitions["BoundedExtras"]
	require.NotNil(t, bounded.AdditionalProperties)
	assert.True(t, bounded.AdditionalProperties.Allows)
	require.NotNil(t, bounded.MaxProperties)
	assert.EqualValues(t, 10, *bounded.MaxProperties)

	// precedence: swagger:type string wins, additionalProperties dropped + warned.
	contra := doc.Definitions["Contradiction"]
	assert.True(t, contra.Type.Contains("string"))
	assert.Nil(t, contra.AdditionalProperties, "additionalProperties dropped on a non-object")

	var shape []grammar.Diagnostic
	for _, d := range diags {
		if d.Code == grammar.CodeShapeMismatch {
			shape = append(shape, d)
		}
	}
	require.Len(t, shape, 1, "exactly one precedence diagnostic")
	assert.Contains(t, shape[0].Message, "only valid on an object")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_additional_properties.json")
}
