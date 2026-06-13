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

// TestCoverage_ObjectKeywordsContext locks the context gating of the
// object validation keywords (minProperties / maxProperties /
// patternProperties):
//
//   - kept on an object-typed model;
//   - stripped + CodeShapeMismatch on a non-object (scalar) model;
//   - dropped + CodeUnsupportedInSimpleSchema on a SimpleSchema param.
func TestCoverage_ObjectKeywordsContext(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/object-keywords-context/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Object model: all three keywords kept.
	obj, ok := doc.Definitions["ObjectModel"]
	require.True(t, ok)
	require.NotNil(t, obj.MinProperties)
	require.NotNil(t, obj.MaxProperties)
	assert.Equal(t, int64(1), *obj.MinProperties)
	assert.Equal(t, int64(9), *obj.MaxProperties)
	assert.Contains(t, obj.PatternProperties, "^x-")

	// Scalar model: all three stripped by the decl-shape re-check.
	scalar, ok := doc.Definitions["ScalarModel"]
	require.True(t, ok)
	assert.Equal(t, []string{"string"}, []string(scalar.Type))
	assert.Nil(t, scalar.MinProperties, "minProperties stripped on non-object model")
	assert.Nil(t, scalar.MaxProperties, "maxProperties stripped on non-object model")
	assert.Empty(t, scalar.PatternProperties, "patternProperties stripped on non-object model")

	// SimpleSchema param: stays a clean string (object keywords dropped).
	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "string", op.Parameters[0].Type)

	// Diagnostics: 3 shape-mismatch (scalar model) + 3 unsupported-in-
	// simple-schema (query param), one per object keyword each.
	var shape, simple int
	for _, d := range diags {
		switch d.Code {
		case grammar.CodeShapeMismatch:
			shape++
		case grammar.CodeUnsupportedInSimpleSchema:
			simple++
		default:
			// other diagnostics are not under test here
		}
	}
	assert.Equal(t, 3, shape, "one shape-mismatch per object keyword on the scalar model")
	assert.Equal(t, 3, simple, "one unsupported-in-simple-schema per object keyword on the query param")

	scantest.CompareOrDumpJSON(t, doc, "object_keywords_context.json")
}
