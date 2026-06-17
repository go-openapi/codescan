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

// TestCoverage_Bug1117 documents the answer to go-swagger issue #1117 ("how to
// define response as array type"): give the swagger:response an in:body field of
// a slice type; the response schema becomes {type: array, items: {$ref}}.
//
// 📖 Need doc: array-body response recipe (in:body Body []T).
func TestCoverage_Bug1117(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1117/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	resp, ok := doc.Responses["arrayResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	require.Len(t, resp.Schema.Type, 1)
	assert.Equal(t, "array", resp.Schema.Type[0])
	require.NotNil(t, resp.Schema.Items)
	assert.Equal(t, "#/definitions/Item", resp.Schema.Items.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_1117_schema.json")
}
