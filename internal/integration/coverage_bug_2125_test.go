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

// TestCoverage_Bug2125 locks go-swagger issue #2125 ("parsing meta info comments
// can parse fields wrong"): markdown content in a swagger:meta block (headings,
// prose mentioning "Api-Version") is kept in the info description and does NOT
// derail field parsing — the real `Version:` field is read correctly.
func TestCoverage_Bug2125(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2125/..."}, WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc.Info)
	assert.Equal(t, "1.0.0", doc.Info.Version, "the real Version field is parsed, not the markdown (go-swagger#2125)")
	assert.True(t, strings.Contains(doc.Info.Description, "# Authentication"), "markdown is kept in the description")
	scantest.CompareOrDumpJSON(t, doc, "bugs_2125_schema.json")
}
