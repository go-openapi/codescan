// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

// TestCoverage_Bug1109 documents the behaviour behind go-swagger issue #1109 ("generates invalid
// swagger.json when returning response is model").
//
// Per @casualjim this is expected behaviour, not a codegen bug: `200: <name>` resolves a
// swagger:response by name, and a route that references a swagger:model by that name needs -m
// (ScanModels) for the model to be discovered.
// The original invalid-spec symptom (a dangling `$ref: #/responses/order`) no longer occurs.
//
// This test pins the two working forms (both with -m):
//
//   - the OP's `200: order` form, resolved via codescan's
//     definition-fallback (model name promoted to a body $ref), and
//   - casualjim's explicit `200: body:order` form.
//
// It is flagged "Need doc" in the backlog: the doc site should explain the `-m` requirement and the
// swagger:response-vs-model name resolution.
func TestCoverage_Bug1109(t *testing.T) {
	t.Run("OP form `200: order` resolves via definition-fallback with -m", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./bugs/1109/responseref/..."},
			WorkDir:    scantest.FixturesDir(),
			ScanModels: true,
		})
		require.NoError(t, err)
		require.NotNil(t, doc)

		assertOrderBodyResponse(t, doc)
		scantest.CompareOrDumpJSON(t, doc, "bugs_1109_responseref_schema.json")
	})

	t.Run("casualjim form `200: body:order` with -m", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./bugs/1109/bodyref/..."},
			WorkDir:    scantest.FixturesDir(),
			ScanModels: true,
		})
		require.NoError(t, err)
		require.NotNil(t, doc)

		assertOrderBodyResponse(t, doc)
		scantest.CompareOrDumpJSON(t, doc, "bugs_1109_bodyref_schema.json")
	})
}

// assertOrderBodyResponse checks the 200 response carries a body schema $ref to the order
// definition, and that the definition is emitted — i.e. no dangling #/responses/order and no
// missing definitions block.
func assertOrderBodyResponse(t *testing.T, doc *oaispec.Swagger) {
	t.Helper()

	require.Contains(t, doc.Definitions, "order", "the model definition must be emitted under -m")

	pi, ok := doc.Paths.Paths["/orders/{id}"]
	require.True(t, ok)
	require.NotNil(t, pi.Put)
	require.NotNil(t, pi.Put.Responses)

	resp, ok := pi.Put.Responses.StatusCodeResponses[200]
	require.True(t, ok, "the 200 response must be present")
	assert.Empty(t, resp.Ref.String(), "must not emit a dangling #/responses ref")
	require.NotNil(t, resp.Schema, "the 200 response must carry a body schema")
	assert.Equal(t, "#/definitions/order", resp.Schema.Ref.String())
}
