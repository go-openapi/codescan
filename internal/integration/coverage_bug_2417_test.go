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

// TestCoverage_Bug2417 covers go-swagger issue #2417 ("embedding of aliased type").
//
// Embedding an anonymous defined-type whose underlying struct lives in a *different* package than
// the type itself (a.AnotherPackageAlias, underlying color.Color) used to promote no fields — the
// definition came out a bare empty object — because field-AST resolution searched only the
// embedding decl's source file.
//
// The field-promotion path now falls back to the field's own source file (ScanCtx.FileForPos), so
// the fields are promoted.
//
// The other three matrix cells were already correct and are asserted here as
// guard rails against regression:
//   - color.Color embedded            -> "hue" promoted
//   - color.SamePackageAlias embedded -> "hue" promoted
//   - named a.AnotherPackageAlias      -> $ref
func TestCoverage_Bug2417(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2417/b/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Guard rails: the cells that already work.
	assert.Contains(t, doc.Definitions["PlainEmbed"].Properties, "hue",
		"embedding a plain cross-package type promotes its fields")
	assert.Contains(t, doc.Definitions["SameAliasEmbed"].Properties, "hue",
		"embedding a same-package alias promotes its fields")
	named := doc.Definitions["NamedCrossAlias"].Properties["anyName"]
	assert.Equal(t, "#/definitions/AnotherPackageAlias", named.Ref.String(),
		"a named cross-package alias field is referenced")

	// The #2417 defect: embedding an alias-of-cross-package-type promotes nothing.
	// Expected behaviour is that "hue" is promoted, like every other embed form above.
	assert.Contains(t, doc.Definitions["CrossAliasEmbed"].Properties, "hue",
		"embedding an alias whose underlying type is in another package must "+
			"promote its fields (go-swagger#2417)")
}
