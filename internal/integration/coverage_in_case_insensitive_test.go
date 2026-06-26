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

// TestCoverage_InCaseInsensitive witnesses Q29 — the `in:` value must be normalised
// case-insensitively so go-swagger-generated `in: Body` lands at the body location, not silently
// degrading to the `query` default (parameters) or producing an invalidIn diagnostic (responses).
//
// Pre-fix shape (silent miscategorisation):
//
//	mixedCaseParamsRequest:
//	  parameters: every field at "in: query" because strict-case
//	  lookup against validParamIn rejected Body/QUERY/Path/Header/
//	  FORMDATA.
//
// Post-fix shape (one parameter per declared location):
//
//	parameters: [body, query, path, header, formData]
//	mixedCaseResponse: shaped as a body response with Payload
//	schema.
func TestCoverage_InCaseInsensitive(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/in-case-insensitive/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.Paths.Paths, "/mixed")
	op := doc.Paths.Paths["/mixed"].Post
	require.NotNil(t, op, "POST /mixed operation must exist")
	locs := make(map[string]int)
	for _, p := range op.Parameters {
		locs[p.In]++
	}
	assert.Equal(t, 1, locs["body"], "in: Body must canonicalise to body")
	assert.Equal(t, 1, locs["query"], "in: QUERY must canonicalise to query")
	assert.Equal(t, 1, locs["path"], "in: Path must canonicalise to path")
	assert.Equal(t, 1, locs["header"], "in: Header must canonicalise to header")
	assert.Equal(t, 1, locs["formData"], "in: FORMDATA must canonicalise to formData")

	require.Contains(t, doc.Responses, "mixedCaseResponse")
	resp := doc.Responses["mixedCaseResponse"]
	assert.NotNil(t, resp.Schema, "mixed-case in: Body must route through body branch — schema present")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_in_case_insensitive.json")
}
