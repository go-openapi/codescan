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

// TestCoverage_ResponseBodyFieldExample locks example: on a swagger:response
// struct Body field — the body-field keywords land on the body schema rather
// than the discarded header (go-swagger#3013, #2942 family).
func TestCoverage_ResponseBodyFieldExample(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/response-bodyfield-example/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	arr, ok := doc.Responses["ArrayBodyResp"]
	require.True(t, ok)
	require.NotNil(t, arr.Schema)
	assert.Equal(t, []string{"array"}, []string(arr.Schema.Type))
	assert.Equal(t, []any{"a", "b"}, arr.Schema.Example)

	scalar, ok := doc.Responses["ScalarBodyResp"]
	require.True(t, ok)
	require.NotNil(t, scalar.Schema)
	assert.Equal(t, []string{"string"}, []string(scalar.Schema.Type))
	assert.Equal(t, "hello", scalar.Schema.Example)

	scantest.CompareOrDumpJSON(t, doc, "response_bodyfield_example.json")
}
