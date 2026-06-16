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

// TestCoverage_Bug1092 locks go-swagger issue #1092 ("mapping values are not
// allowed in this context"): a swagger:meta block with colon-bearing prose
// (host:port, ratios, URLs) parses cleanly — the prose stays in the description
// and the real Version field is read; no YAML "mapping values" error.
func TestCoverage_Bug1092(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1092/..."}, WorkDir: scantest.FixturesDir(),
	})
	require.NoError(t, err, "colon-bearing meta prose must not raise a YAML mapping error (go-swagger#1092)")
	require.NotNil(t, doc.Info)
	assert.Equal(t, "1.0.0", doc.Info.Version)
	assert.True(t, strings.Contains(doc.Info.Description, "host:port"))
	scantest.CompareOrDumpJSON(t, doc, "bugs_1092_schema.json")
}
