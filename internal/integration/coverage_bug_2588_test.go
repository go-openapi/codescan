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

// TestCoverage_Bug2588 locks go-swagger issue #2588 ("panic on parsing interface type definition"):
// a struct embedding a named type whose underlying type is an interface (type B A; type C struct{ B
// }) no longer panics with "interface conversion: ast.Expr is *ast.Ident, not *ast.InterfaceType".
func TestCoverage_Bug2588(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2588/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err, "scanning the embedded interface-alias must not panic (go-swagger#2588)")

	_, ok := doc.Definitions["C"]
	assert.True(t, ok, "the embedding struct is emitted")
	b := doc.Definitions["B"]
	assert.Equal(t, "#/definitions/A", b.Ref.String(), "the interface-alias resolves to a $ref")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2588_schema.json")
}
