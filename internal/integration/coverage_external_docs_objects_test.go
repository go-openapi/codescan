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

// TestCoverage_ExternalDocsObjects locks externalDocs on non-meta objects:
// emitted on swagger:route + swagger:operation operations and on a
// swagger:model schema, and rejected (with a diagnostic) on a simple-schema
// query parameter.
func TestCoverage_ExternalDocsObjects(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/external-docs-objects/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// swagger:route operation.
	route := doc.Paths.Paths["/route"].Get
	require.NotNil(t, route)
	require.NotNil(t, route.ExternalDocs)
	assert.Equal(t, "route docs", route.ExternalDocs.Description)
	assert.Equal(t, "https://route.example.org", route.ExternalDocs.URL)

	// swagger:operation operation (already worked via YAML — locked here).
	op := doc.Paths.Paths["/operation"].Get
	require.NotNil(t, op)
	require.NotNil(t, op.ExternalDocs)
	assert.Equal(t, "https://operation.example.org", op.ExternalDocs.URL)

	// swagger:model schema.
	model, ok := doc.Definitions["Model"]
	require.True(t, ok)
	require.NotNil(t, model.ExternalDocs)
	assert.Equal(t, "model docs", model.ExternalDocs.Description)
	assert.Equal(t, "https://model.example.org", model.ExternalDocs.URL)

	// Simple-schema query param: dropped + diagnostic, param stays clean.
	list := doc.Paths.Paths["/list"].Get
	require.NotNil(t, list)
	require.Len(t, list.Parameters, 1)
	assert.Equal(t, "query", list.Parameters[0].In)
	var rejected bool
	for _, d := range diags {
		if d.Code == grammar.CodeUnsupportedInSimpleSchema {
			assert.Contains(t, d.Message, "externalDocs")
			rejected = true
		}
	}
	assert.True(t, rejected, "externalDocs on a simple-schema param should be rejected with a diagnostic")

	scantest.CompareOrDumpJSON(t, doc, "external_docs_objects.json")
}
