// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	"github.com/go-openapi/codescan"
)

// TestCoverage_HeaderExtensions pins M2's Walker.Extension wiring on
// the response-header path. Pre-M2, user-authored `Extensions:`
// blocks on header fields were silently dropped; post-M2 they land
// on the header's Extensions map with grammar-typed values.
func TestCoverage_HeaderExtensions(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/header-extensions/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Response is registered at the doc level and the operation
	// references it via $ref — the path-level operation has no
	// inline headers, so inspect the top-level definition.
	require.Contains(t, doc.Responses, "extendedResponse")
	resp := doc.Responses["extendedResponse"]

	require.Contains(t, resp.Headers, "X-Rate-Limit")
	hdr := resp.Headers["X-Rate-Limit"]

	// Extensions present on the header. The YAML body of the
	// Extensions block was parsed via grammar's typed-extensions
	// pipeline, so the values arrive pre-typed (string + bool).
	val, ok := hdr.Extensions["x-rate-window"]
	require.True(t, ok, "x-rate-window extension should be present on header")
	assert.Equal(t, "60s", val)

	val, ok = hdr.Extensions["x-burst-allowed"]
	require.True(t, ok, "x-burst-allowed extension should be present on header")
	assert.Equal(t, true, val)
}
