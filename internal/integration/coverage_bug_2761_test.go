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

// TestCoverage_Bug2761 documents the resolution to go-swagger issue #2761 ("allOf in response
// doesn't use $ref"): a response body with an embedded `swagger:allOf` member emits a proper
// `allOf: [{$ref}, …]` — provided the embedded base is a swagger:model (a definition to
// reference).
//
// The reporter embedded a swagger:*response* (which has no definition), so its fields were inlined;
// using a swagger:model yields the $ref, as locked here.
//
// 📖 Need doc: an allOf $ref requires the embedded base to be a swagger:model, not a
// swagger:response.
func TestCoverage_Bug2761(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2761/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["userResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	require.Len(t, resp.Schema.AllOf, 2, "the embedded model is an allOf arm; the inline fields are the other")
	assert.Equal(t, "#/definitions/BaseResponse", resp.Schema.AllOf[0].Ref.String(),
		"the embedded swagger:model base must be referenced via $ref, not inlined")
	assert.Contains(t, resp.Schema.AllOf[1].Properties, "user")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2761_schema.json")
}
