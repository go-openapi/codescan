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

// TestCoverage_Bug3213 locks the behaviour behind go-swagger issue #3213 ("Consider TypeSpec
// comments"): doc comments and swagger annotations that attach to an individual TypeSpec inside a
// grouped `type ( ... )` declaration — not to the enclosing GenDecl — are parsed.
//
// The proof is two models in ONE type group, each carrying its OWN doc comment: both surface their
// distinct titles, which a GenDecl-only reader could not produce.
// A grouped `swagger:enum` referenced by one of them also has its values parsed.
func TestCoverage_Bug3213(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3213/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	alpha, ok := doc.Definitions["Alpha"]
	require.True(t, ok, "grouped model Alpha must be emitted")
	assert.Equal(t, "Alpha is the first grouped model.", alpha.Title,
		"per-TypeSpec doc must be captured")

	beta, ok := doc.Definitions["Beta"]
	require.True(t, ok, "grouped model Beta must be emitted")
	assert.Equal(t, "Beta is the second grouped model, sharing Alpha's type group.", beta.Title,
		"the second TypeSpec in the same group must keep its own distinct doc")

	// The grouped swagger:enum is parsed: its values reach the referencing field.
	require.Contains(t, beta.Properties, "status")
	assert.Equal(t, []any{"on", "off"}, beta.Properties["status"].Enum,
		"a swagger:enum declared in a type group must have its const values parsed")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3213_schema.json")
}
