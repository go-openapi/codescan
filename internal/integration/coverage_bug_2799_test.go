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

// TestCoverage_Bug2799 locks the fix for go-swagger issue #2799 ("swagger
// generator should keep the format of the API description"): a markdown bullet
// list in the description used to lose its leading `-` (silent strip). The
// grammar2 parser preserves it (forthcoming-features §4.2), so the meta
// description round-trips the list verbatim.
//
// (Distinct from #3211, which is about markdown *table* rows losing their
// leading pipe — still open.)
func TestCoverage_Bug2799(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2799/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Info)

	assert.Equal(t, "- keep markdown list\n- in the description", doc.Info.Description,
		"the markdown bullet list must be preserved verbatim in the description")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2799_schema.json")
}
