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

// TestCoverage_EmbedInheritance locks the generalized "annotations on an
// embedded field apply to the members it promotes" rule (go-swagger#2701)
// beyond parameters: a `required:` annotation on an embed propagates to the
// promoted properties of a model schema and of a response body schema (built
// through the same schema builder). Exportedness stays per-field — the
// unexported promoted member never surfaces.
func TestCoverage_EmbedInheritance(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/embed-inheritance/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Model schema: required: on the embed propagates to promoted id/name;
	// the normal Extra field stays optional; unexported hidden never appears.
	widget := doc.Definitions["Widget"]
	assert.Contains(t, widget.Properties, "id")
	assert.Contains(t, widget.Properties, "name")
	assert.Contains(t, widget.Properties, "extra")
	assert.NotContains(t, widget.Properties, "hidden")
	assert.ElementsMatch(t, []string{"id", "name"}, widget.Required,
		"required: on the embed must propagate to the promoted properties (go-swagger#2701)")

	// Response body: a named struct body is emitted as a discovered
	// definition and $ref'd. The same rule applies through the schema
	// builder, so the body definition carries the inherited required list.
	resp, ok := doc.Responses["widgetResponse"]
	require.True(t, ok)
	require.NotNil(t, resp.Schema)
	assert.Equal(t, "#/definitions/envelope", resp.Schema.Ref.String())

	envelope := doc.Definitions["envelope"]
	assert.ElementsMatch(t, []string{"id", "name"}, envelope.Required,
		"required: on an embed inside a response body must propagate too (go-swagger#2701)")
	assert.NotContains(t, envelope.Properties, "hidden")

	// Responses builder's own embed recursion: an embedded struct promotes
	// its fields as headers (in: header, the default), confirming the path is
	// intact.
	hdrs, ok := doc.Responses["embeddedHeadersResponse"]
	require.True(t, ok)
	assert.Contains(t, hdrs.Headers, "X-Request-Id")
	assert.Contains(t, hdrs.Headers, "X-Count")

	// Distinguishing case: `in: body` on the embed routes the promoted field
	// to the response body (it would otherwise default to a header) — proving
	// the responses builder inherits in: from the embed (go-swagger#2701).
	body, ok := doc.Responses["embeddedBodyResponse"]
	require.True(t, ok)
	require.NotNil(t, body.Schema)
	assert.True(t, body.Schema.Type.Contains("string"),
		"in: body on the embed must make the promoted field the response body")

	scantest.CompareOrDumpJSON(t, doc, "embed_inheritance.json")
}
