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

// TestCoverage_ResponseNameKeyword locks the `name:` keyword on a
// swagger:response struct field: it renames the response header (the
// Headers map key), overriding the json-tag / Go-field derivation, and —
// being a structural keyword — is stripped from the header description
// rather than leaking into it as prose. Mirrors the parameter-side
// TestCoverage_ParamNameKeyword.
func TestCoverage_ResponseNameKeyword(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-name-keyword/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["widgetResponse"]
	require.True(t, ok)

	// The header is keyed by the name: keyword, not the Go field name.
	require.NotContains(t, resp.Headers, "RateLimit", "the Go field name must not survive when name: renames the header")
	hdr, ok := resp.Headers["X-Rate-Limit"]
	require.True(t, ok, "the name: keyword renames the response header (map key)")
	assert.Equal(t, "requests remaining in the window", hdr.Description, "name: / minimum: must not leak into the description")
	require.NotNil(t, hdr.Minimum)
	assert.Equal(t, float64(0), *hdr.Minimum, "validation keywords still apply alongside name:")

	// The body field still produces the response schema.
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "#/definitions/Widget", resp.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_name_keyword.json")
}
