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

// TestCoverage_Bug2396 asserts the EXPECTED resolution of go-swagger issue #2396 ("model enum
// recognition and handling spaces").
//
// Currently RED.
//
// The bracketed `enum: [a, b, c]` form must produce the same enum as the unbracketed `enum: a, b,
// c` form — the surrounding `[`/`]` are delimiters and must be stripped.
// The scanner currently glues them onto the first/last value (["[issues", "pulls", "projects]"]).
// The unbracketed form (which already trims correctly) is the green guard rail.
func TestCoverage_Bug2396(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2396/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	want := []any{"issues", "pulls", "projects"}

	// Guard rail: unbracketed form already correct.
	assert.Equal(t, want, doc.Definitions["Spaced"].Properties["kind"].Enum)

	// The #2396 defect: bracketed form must strip the surrounding brackets.
	assert.Equal(t, want, doc.Definitions["Bracket"].Properties["kind"].Enum,
		"the `enum: [a, b, c]` brackets must be stripped (go-swagger#2396)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2396_schema.json")
}
