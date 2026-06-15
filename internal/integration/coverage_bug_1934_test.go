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

// TestCoverage_Bug1934 locks go-swagger issue #1934 ("model declared in function
// not picked up"): a swagger:model and a swagger:parameters declared on types
// INSIDE a function body are both discovered — the model resolves as the route's
// response body $ref, and the params type contributes the query parameter.
func TestCoverage_Bug1934(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1934/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	_, ok := doc.Definitions["someResponse"]
	require.True(t, ok, "a swagger:model declared inside a function must be discovered")

	op := doc.Paths.Paths["/pets"].Get
	require.NotNil(t, op)
	def := op.Responses.Default
	require.NotNil(t, def)
	require.NotNil(t, def.Schema)
	assert.Equal(t, "#/definitions/someResponse", def.Schema.Ref.String())
	require.Len(t, op.Parameters, 1)
	assert.Equal(t, "foo", op.Parameters[0].Name)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1934_schema.json")
}
