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

// TestCoverage_Bug2942 verifies the fix for go-swagger issue #2942 ("simple
// string response"). Two paths to a primitive response body now work:
//   - `200: body:string` in a swagger:route emits {schema: {type: string}}
//     (the unambiguous `body:` tag; the bare `200: string` form stays
//     unsupported on purpose — see TestCoverage_PrimitiveResponse);
//   - a swagger:response whose Body is a primitive `string` now emits
//     {schema: {type: string}} instead of no schema (Typed routed to the body
//     schema rather than a discarded header).
func TestCoverage_Bug2942(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2942/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// `200: body:string` now emits a typed schema.
	v := doc.Paths.Paths["/version"]
	require.NotNil(t, v.Get)
	require.NotNil(t, v.Get.Responses)
	r200, has := v.Get.Responses.StatusCodeResponses[200]
	require.True(t, has, "`200: body:string` should emit a response")
	require.NotNil(t, r200.Schema)
	assert.Equal(t, []string{"string"}, []string(r200.Schema.Type))

	// The swagger:response string-body workaround now carries a schema.
	resp, ok := doc.Responses["versionResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema, "primitive `Body string` should emit schema {type: string}")
	assert.Equal(t, []string{"string"}, []string(resp.Schema.Type))

	scantest.CompareOrDumpJSON(t, doc, "bugs_2942_schema.json")
}
