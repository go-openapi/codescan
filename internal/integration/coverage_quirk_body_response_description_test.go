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

// TestQuirk_BodyResponseDescription verifies the fix for quirk G1 (doc-site-quirks.md): a `body:`
// response with no trailing description no longer fills response.description with the bare Go type
// token.
//
// Instead the description is derived from the referenced model's godoc, falling back to the HTTP
// status reason phrase, then a neutral placeholder.
//
//	200: body:Pet      → "Pet is a documented model used as a response body."
//	404: body:Blank    → "Not Found"      (model has no godoc → status phrase)
//	500: body:string   → "Internal Server Error" (primitive → status phrase)
//	default: body:string → "Default response"    (no phrase → placeholder)
func TestQuirk_BodyResponseDescription(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./quirks/body-response-description/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	op := doc.Paths.Paths["/quirk"].Get
	require.NotNil(t, op)
	codes := op.Responses.StatusCodeResponses

	// Tier 1: documented model → its godoc (title), not the token "Pet".
	require.Contains(t, codes, 200)
	assert.Equal(t, "Pet is a documented model used as a response body.", codes[200].Description)
	assert.Equal(t, "#/definitions/Pet", codes[200].Schema.Ref.String())

	// Tier 2: model with no godoc → HTTP status reason phrase, not "Blank".
	require.Contains(t, codes, 404)
	assert.Equal(t, "Not Found", codes[404].Description)
	assert.Equal(t, "#/definitions/Blank", codes[404].Schema.Ref.String())

	// Tier 2: primitive body → HTTP status reason phrase, not "string".
	require.Contains(t, codes, 500)
	assert.Equal(t, "Internal Server Error", codes[500].Description)
	assert.Equal(t, "string", codes[500].Schema.Type[0])

	// Tier 3: `default` code has no reason phrase → neutral placeholder.
	require.NotNil(t, op.Responses.Default)
	assert.Equal(t, "Default response", op.Responses.Default.Description)

	// In no case does the description echo the Go type token.
	for _, leaked := range []string{"Pet", "Blank", "string"} {
		for code, resp := range codes {
			assert.NotEqual(t, leaked, resp.Description,
				"response %d description must not echo the type token", code)
		}
	}

	scantest.CompareOrDumpJSON(t, doc, "quirk_body_response_description.json")
}
