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

// TestCoverage_AdditionalProperties_SimpleSchemaSafeguard proves the safeguard:
// additionalProperties and patternProperties (object-schema keywords) cannot
// land on an OAS2 SimpleSchema produced by parameters.Builder (a non-body
// parameter) or responses.Builder (a response header). Each occurrence is
// dropped with a CodeUnsupportedInSimpleSchema diagnostic — both keywords are
// declared CtxSchema-only, so handlers.isFullSchemaOnly routes them to
// UnsupportedSimpleSchemaString in the param/header/items dispatchers.
func TestCoverage_AdditionalProperties_SimpleSchemaSafeguard(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/additional-properties-simpleschema/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The query parameter stays a clean string (the object keywords have no
	// SimpleSchema slot to land in — Parameter embeds SimpleSchema only).
	op := doc.Paths.Paths["/safeguard"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "query", op.Parameters[0].In)
	assert.Equal(t, "string", op.Parameters[0].Type)

	// The response header likewise stays a clean string (the response is a
	// shared definition referenced by $ref from the operation).
	resp, ok := doc.Responses["safeguardResponse"]
	require.True(t, ok)
	hdr, ok := resp.Headers["X-Thing"]
	require.True(t, ok)
	assert.Equal(t, "string", hdr.Type)

	// Two diagnostics per site (additionalProperties + patternProperties) × two
	// sites (param + header) = four; all CodeUnsupportedInSimpleSchema.
	var simple int
	for _, d := range diags {
		if d.Code == grammar.CodeUnsupportedInSimpleSchema {
			simple++
			assert.Contains(t, d.Message, "full-Schema-only")
		}
	}
	assert.Equal(t, 4, simple, "additionalProperties + patternProperties dropped on both the param and the header")
}
