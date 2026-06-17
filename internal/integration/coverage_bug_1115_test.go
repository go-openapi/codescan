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

// TestCoverage_Bug1115 locks the fix for go-swagger issue #1115 ("Missing parser
// for a *ast.StarExpr"): a type aliased to a POINTER (type BarResponse *Thing)
// used to warn and be omitted. The grammar2 scanner handles the StarExpr — the
// pointer is dereferenced and the alias emits a $ref like the non-pointer form.
func TestCoverage_Bug1115(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1115/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	bar, ok := doc.Definitions["barResponse"]
	require.True(t, ok, "the pointer alias is not omitted")
	assert.Equal(t, "#/definitions/Thing", bar.Ref.String(),
		"a pointer alias dereferences to a $ref, like the non-pointer alias")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1115_schema.json")
}
