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

// TestCoverage_Bug2305 locks go-swagger issue #2305 ("query params, enum
// dropdown"): a query parameter with an enum produces a clean, valid parameter —
// the enum is present, with no illegal `schema` and no duplicated description.
func TestCoverage_Bug2305(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2305/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	p := doc.Paths.Paths["/items"].Get.Parameters[0]
	assert.Equal(t, "query", p.In)
	assert.Equal(t, "sort", p.Name)
	assert.Equal(t, []any{"asc", "desc"}, p.Enum)
	assert.Equal(t, "Sort order", p.Description)
	assert.Nil(t, p.Schema, "a non-body param must not carry a schema")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2305_schema.json")
}
