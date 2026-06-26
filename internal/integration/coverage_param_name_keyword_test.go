// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_ParamNameKeyword locks the `name:` keyword on a swagger:parameters struct field: it
// renames the JSON parameter name, overriding the json-tag / Go-field derivation, and — being a
// structural keyword — is stripped from the parameter description rather than leaking into it as
// prose (it previously had no handler and fell through to Block.Prose()).
func TestCoverage_ParamNameKeyword(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/param-name-keyword/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	op := doc.Paths.Paths["/widgets"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 2)

	byIn := map[string]oaispec.Parameter{}
	for _, p := range op.Parameters {
		byIn[p.In] = p
	}

	body, ok := byIn["body"]
	require.True(t, ok)
	assert.Equal(t, "widget", body.Name, "the name: keyword renames the body param (was Go field name `Body`)")
	assert.Equal(t, "the widget payload", body.Description, "name: must not leak into the description")
	require.NotNil(t, body.Schema)
	assert.Equal(t, "#/definitions/Widget", body.Schema.Ref.String())
	assert.Equal(t, "Body", body.Extensions["x-go-name"], "the Go field name is preserved as x-go-name")

	query, ok := byIn["query"]
	require.True(t, ok)
	assert.Equal(t, "count", query.Name, "the name: keyword renames the query param (was `Quantity`)")
	assert.Equal(t, "how many to make", query.Description, "name: / minimum: must not leak into the description")
	require.NotNil(t, query.Minimum)
	assert.Equal(t, float64(1), *query.Minimum, "validation keywords still apply alongside name:")
	assert.Equal(t, "Quantity", query.Extensions["x-go-name"])

	scantest.CompareOrDumpJSON(t, doc, "enhancements_param_name_keyword.json")
}
