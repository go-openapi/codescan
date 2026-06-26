// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_InnerMarkdown exercises the swagger:description | literal block scalar end to end: a
// model and a field description carry verbatim markdown (table with leading pipes, significant
// blank lines, indented ordered list) into the emitted spec.
//
// Reframes go-swagger#3211.
func TestCoverage_InnerMarkdown(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/inner-markdown/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.Definitions, "Widget")
	w := doc.Definitions["Widget"]

	// Title still comes from the godoc preamble; the markdown is the description.
	assert.Equal(t, "Widget is a thing.", w.Title)

	// Table leading pipes survive (the literal #3211 grievance), the blank line between paragraphs is
	// preserved, and the marker never leaks.
	assert.Contains(t, w.Description, "| name | purpose |")
	assert.Contains(t, w.Description, "**markdown**")
	assert.Contains(t, w.Description, "\n\n", "significant blank line preserved")
	assert.NotContains(t, w.Description, "swagger:description")
	assert.False(t, strings.HasPrefix(w.Description, "|"), "the | marker must not leak into the body")

	// Field description keeps indentation of the ordered list.
	name := w.Properties["name"]
	assert.Contains(t, name.Description, "The name must be:")
	assert.Contains(t, name.Description, "  1. unique")
	assert.Contains(t, name.Description, "  2. lowercase")

	scantest.CompareOrDumpJSON(t, doc, "enhancements_inner_markdown.json")
}
