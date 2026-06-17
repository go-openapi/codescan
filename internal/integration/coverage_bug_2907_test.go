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

// TestCoverage_Bug2907 locks the fix for go-swagger issue #2907 ("Go-Swagger
// not generating properties in yaml file"): a response whose body is a slice
// of a cross-package model (`Body []data.Movie`) used to emit `items: {}` —
// the model's properties were lost. The cross-package model is now resolved,
// so the response schema is an array of `$ref` to the emitted definition.
func TestCoverage_Bug2907(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2907/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["moviesResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "array", resp.Schema.Type[0])
	require.NotNil(t, resp.Schema.Items)
	require.NotNil(t, resp.Schema.Items.Schema)
	assert.Equal(t, "#/definitions/Movie", resp.Schema.Items.Schema.Ref.String(),
		"the cross-package item model must resolve to a $ref, not an empty items")

	movie, ok := doc.Definitions["Movie"]
	require.True(t, ok, "the cross-package model must be emitted")
	assert.Contains(t, movie.Properties, "title")
	assert.Contains(t, movie.Properties, "genres")
	assert.NotContains(t, movie.Properties, "CreatedAt", "json:\"-\" field must be excluded")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2907_schema.json")
}
