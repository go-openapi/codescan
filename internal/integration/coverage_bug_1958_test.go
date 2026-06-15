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

// TestCoverage_Bug1958 locks go-swagger issue #1958 ("Ability to skip embedded
// tags"): a vendor extension in the swagger:operation YAML body is preserved in
// full — including nested keys that share a name with a known keyword (here,
// `responses:` inside x-amazon-apigateway-integration), which must NOT be
// confused with the operation's own responses.
func TestCoverage_Bug1958(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1958/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/aws"].Get
	require.NotNil(t, op)

	// The operation's own responses are intact.
	_, ok := op.Responses.StatusCodeResponses[200]
	assert.True(t, ok, "the operation's own 200 response is intact")

	// The vendor extension is preserved with its nested map (incl. responses).
	ext, ok := op.Extensions["x-amazon-apigateway-integration"]
	require.True(t, ok, "the vendor extension must be preserved")
	m, ok := ext.(map[string]any)
	require.True(t, ok, "the extension value must be a map, not null")
	assert.Equal(t, "http_proxy", m["type"])
	assert.Contains(t, m, "responses", "the nested responses key inside the extension survives")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1958_schema.json")
}
