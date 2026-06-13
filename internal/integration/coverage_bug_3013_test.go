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

// TestCoverage_Bug3013 verifies the fix for go-swagger issue #3013: an
// `example:` on a swagger:response whose body is a top-level array type now
// lands on the response body schema (it was previously dropped — the response
// decl comment only contributed the description).
func TestCoverage_Bug3013(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/3013/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["getNtpServersResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	require.NotEmpty(t, resp.Schema.Type)
	assert.Equal(t, "array", resp.Schema.Type[0])

	// FIXED: the array example is carried onto the response schema.
	assert.Equal(t, []any{"10.10.10.10", "20.20.20.20"}, resp.Schema.Example)

	scantest.CompareOrDumpJSON(t, doc, "bugs_3013_schema.json")
}
