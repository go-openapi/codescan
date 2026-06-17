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

// TestCoverage_Bug2746 locks the fix for go-swagger issue #2746 ("Strfmt:
// array of UUID"): a slice of a swagger:strfmt-uuid type used to lose the
// format on its items (strfmt only applied to a single value). The format now
// applies to the array's items.
func TestCoverage_Bug2746(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2746/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	prop := doc.Definitions["FilterBy"].Properties["vehicleTypes"]
	assert.Equal(t, "array", prop.Type[0])
	require.NotNil(t, prop.Items)
	require.NotNil(t, prop.Items.Schema)
	assert.Equal(t, "string", prop.Items.Schema.Type[0])
	assert.Equal(t, "uuid", prop.Items.Schema.Format, "strfmt must apply to array items, not just scalars")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2746_schema.json")
}
