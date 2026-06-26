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

// TestCoverage_AdditionalProperties_Field locks the field-level additionalProperties: keyword
// (Phase 2b): override a map element schema, true / typed / model-reference values, the
// allOf-sibling form on a referenced-model field, and the lowest-priority contradiction.
func TestCoverage_AdditionalProperties_Field(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/additional-properties-field/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Holder"].Properties

	// map element schema overridden (string → integer).
	over := props["overriddenMap"].AdditionalProperties
	require.NotNil(t, over)
	require.NotNil(t, over.Schema)
	assert.True(t, over.Schema.Type.Contains("integer"), "map element overridden to integer")

	// true → SchemaOrBool{Allows:true}.
	anyMap := props["anyMap"].AdditionalProperties
	require.NotNil(t, anyMap)
	assert.Nil(t, anyMap.Schema)
	assert.True(t, anyMap.Allows)

	// model reference → $ref value schema.
	ref := props["refMap"].AdditionalProperties
	require.NotNil(t, ref)
	require.NotNil(t, ref.Schema)
	assert.Equal(t, "#/definitions/Inner", ref.Schema.Ref.String())

	// referenced-model field closed via an allOf sibling:
	// {allOf:[{$ref},{additionalProperties:false}]}.
	closed := props["closedInner"]
	require.Len(t, closed.AllOf, 2)
	assert.Equal(t, "#/definitions/Inner", closed.AllOf[0].Ref.String())
	require.NotNil(t, closed.AllOf[1].AdditionalProperties)
	assert.False(t, closed.AllOf[1].AdditionalProperties.Allows)

	// contradiction: string field keeps its type, additionalProperties dropped + warned.
	bad := props["badField"]
	assert.True(t, bad.Type.Contains("string"))
	assert.Nil(t, bad.AdditionalProperties)

	var shape []grammar.Diagnostic
	for _, d := range diags {
		if d.Code == grammar.CodeShapeMismatch {
			shape = append(shape, d)
		}
	}
	require.Len(t, shape, 1, "exactly one contradiction diagnostic")
	assert.Contains(t, shape[0].Message, "only valid on an object")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_additional_properties_field.json")
}
