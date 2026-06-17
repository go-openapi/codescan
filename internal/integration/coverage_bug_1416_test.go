// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug1416 locks the fix for go-swagger issue #1416 ("parameters do
// not get properly linked to operations"): the reporter saw a spurious top-level
// $ref emitted alongside the schema on a body parameter (malformed). The body
// param is now clean — {in: body, schema: {$ref}}, no top-level $ref — and links
// to the operation.
func TestCoverage_Bug1416(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1416/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/api/v1/clusters"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "body", p.In)
	assert.Empty(t, p.Ref.String(), "no spurious top-level $ref on the parameter (the #1416 bug)")
	require.NotNil(t, p.Schema)
	assert.Equal(t, "#/definitions/cluster", p.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_1416_schema.json")
}
