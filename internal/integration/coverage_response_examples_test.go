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

// TestCoverage_ResponseExamplesByMime covers response-level `examples` (a map
// keyed by mime type) on a struct-based `swagger:response` (go-swagger#2871).
//
// The `swagger:operation` YAML path already carries examples for free (see
// coverage_bug_1713 / coverage_bug_2871); this verifies the Go-struct
// `swagger:response` path, where the `examples:` block in the decl comment is
// parsed into the OAS2 Response.examples field.
func TestCoverage_ResponseExamplesByMime(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-examples-by-mime/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp, ok := doc.Responses["widgetResponse"]
	require.TrueT(t, ok, "expected a widgetResponse in the responses section")
	require.NotNil(t, resp.Examples, "response must carry an examples map")

	jsonEx, ok := resp.Examples["application/json"].(map[string]any)
	require.TrueT(t, ok, "expected an application/json example object")
	assert.MapEqualT(t, map[string]any{"name": "alice", "count": float64(3)}, jsonEx)

	xmlEx, ok := resp.Examples["application/xml"]
	require.TrueT(t, ok, "expected an application/xml example")
	assert.EqualT(t, any("<widget><name>alice</name></widget>"), xmlEx)

	// The body schema and description survive alongside the examples.
	assert.EqualT(t, "WidgetResponse returns a widget, with response-level examples per mime type.", resp.Description)
	require.NotNil(t, resp.Schema)

	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_examples_by_mime.json")
}
