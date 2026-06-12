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

// TestCoverage_Bug3100 locks the fix for go-swagger issue #3100
// ("`in: formData` in `swagger:route` annotation translates to nothing").
// A parameter declared with `in: formData` inside an inline swagger:route
// annotation must round-trip its `in` field verbatim; the scanner used to
// only recognise `form` and dropped `in` entirely from the spec.
func TestCoverage_Bug3100(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/3100/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pi, ok := doc.Paths.Paths["/v1/example/route"]
	require.True(t, ok, "the route must be present")
	require.NotNil(t, pi.Post)

	in := make(map[string]string, len(pi.Post.Parameters))
	for _, p := range pi.Post.Parameters {
		assert.NotEmpty(t, p.In, "parameter %q must carry an `in` field", p.Name)
		in[p.Name] = p.In
	}

	// The regression: formData params lost their `in`.
	assert.Equal(t, "formData", in["encryption_public_key"])
	assert.Equal(t, "formData", in["signature"])
	assert.Equal(t, "query", in["verbose"])

	scantest.CompareOrDumpJSON(t, doc, "bugs_3100_schema.json")
}
