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

// TestCoverage_Bug2549 locks go-swagger issue #2549 ("example not working for imported Types"): an
// example on a field whose type is an imported named type (a $ref) is applied via allOf — the
// example is no longer dropped ("shows 0").
//
// (The example value is carried as the string "210000"; numeric coercion of example values is the
// separate forthcoming-features §2.1 / #1268 concern.)
func TestCoverage_Bug2549(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2549/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	mv := doc.Definitions["Property"].Properties["marketValue"]
	require.Len(t, mv.AllOf, 2, "imported-type field with example becomes allOf[$ref, example]")
	assert.Equal(t, "#/definitions/Decimal", mv.AllOf[0].Ref.String())
	assert.Equal(t, "210000", mv.AllOf[1].Example, "the example is applied, not dropped (go-swagger#2549)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2549_schema.json")
}
