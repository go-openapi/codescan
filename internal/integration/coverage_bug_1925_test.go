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

// TestCoverage_Bug1925 locks go-swagger issue #1925 ("parameters do not support
// interface"): a body parameter typed `[]map[string]interface{}` no longer
// aborts spec generation — it produces a valid array-of-object schema with open
// additionalProperties.
func TestCoverage_Bug1925(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1925/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/report"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "body", p.In)
	require.NotNil(t, p.Schema)
	require.Len(t, p.Schema.Type, 1)
	assert.Equal(t, "array", p.Schema.Type[0])
	require.NotNil(t, p.Schema.Items)
	require.NotNil(t, p.Schema.Items.Schema)
	assert.Equal(t, "object", p.Schema.Items.Schema.Type[0])
	assert.NotNil(t, p.Schema.Items.Schema.AdditionalProperties,
		"map[string]interface{} becomes an object with additionalProperties")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1925_schema.json")
}
