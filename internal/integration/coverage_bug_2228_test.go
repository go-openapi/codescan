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

// TestCoverage_Bug2228 locks go-swagger issue #2228 ("how to give empty summary in swagger:route"):
// the auto-derived summary (first prose line) is suppressed by simply omitting that line — a
// route with only `responses:` has no summary.
func TestCoverage_Bug2228(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2228/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	assert.Equal(t, "This becomes the summary.", doc.Paths.Paths["/alpha"].Get.Summary)
	assert.Empty(t, doc.Paths.Paths["/beta"].Get.Summary,
		"omitting the leading prose line yields no summary (go-swagger#2228)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2228_schema.json")
}
