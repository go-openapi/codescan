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

// TestCoverage_ResponseTopLevelExample locks example: on top-level non-struct response bodies: an
// array (example on its own paragraph) and a scalar (trailing example line).
//
// Both land on the response body schema (go-swagger#3013).
func TestCoverage_ResponseTopLevelExample(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/response-toplevel-example/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	arr, ok := doc.Responses["ArrayResp"]
	require.True(t, ok)
	require.NotNil(t, arr.Schema)
	assert.Equal(t, []any{"10.10.10.10", "20.20.20.20"}, arr.Schema.Example)

	scalar, ok := doc.Responses["ScalarResp"]
	require.True(t, ok)
	require.NotNil(t, scalar.Schema)
	assert.Equal(t, "hello", scalar.Schema.Example)

	scantest.CompareOrDumpJSON(t, doc, "response_toplevel_example.json")
}
