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

// TestCoverage_Bug613 locks go-swagger issue #613 ("generating spec of external structs"): a
// referenced struct (here via a []*Ulimit field) is implicitly converted into its own definition
// and referenced by $ref — no unhelpful error.
func TestCoverage_Bug613(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{Packages: []string{"./bugs/613/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true})
	require.NoError(t, err)
	ul := doc.Definitions["Resources"].Properties["ulimits"]
	require.NotNil(t, ul.Items)
	require.NotNil(t, ul.Items.Schema)
	assert.Equal(t, "#/definitions/Ulimit", ul.Items.Schema.Ref.String())
	_, ok := doc.Definitions["Ulimit"]
	assert.True(t, ok, "the referenced external-like struct becomes a definition")
	scantest.CompareOrDumpJSON(t, doc, "bugs_613_schema.json")
}
