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

// TestCoverage_Bug1881 locks go-swagger issue #1881 (the array-of-object
// response part): a swagger:response whose body is a slice of a model produces
// a valid `{type: array, items: {$ref}}` schema.
func TestCoverage_Bug1881(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1881/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["itemsResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	require.Len(t, resp.Schema.Type, 1)
	assert.Equal(t, "array", resp.Schema.Type[0])
	require.NotNil(t, resp.Schema.Items)
	require.NotNil(t, resp.Schema.Items.Schema)
	assert.Equal(t, "#/definitions/Item", resp.Schema.Items.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_1881_schema.json")
}
