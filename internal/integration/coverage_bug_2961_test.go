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

// TestCoverage_Bug2961 locks the fix for go-swagger issue #2961 ("Improper
// parsing of uint enums"): an `enum:` on a uint / uint64 field used to be
// emitted as strings (["1","2","3"]). The values are now coerced to numbers
// against the integer schema type.
func TestCoverage_Bug2961(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2961/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["Sample"].Properties

	ex := props["example"]
	assert.Equal(t, "integer", ex.Type[0])
	assert.Equal(t, []any{1, 2, 3}, ex.Enum,
		"uint enum values must be numbers, not strings")

	ex2 := props["example2"]
	assert.Equal(t, []any{10, 20, 30}, ex2.Enum)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2961_schema.json")
}
