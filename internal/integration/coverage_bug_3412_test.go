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

// TestCoverage_Bug3412 locks the fix for go-swagger issue #3412 ("swagger:enum seems to ignore
// negative integer constants"): Go's scanner never produces a negative numeric literal — `-1` is a
// unary minus applied to the literal `1` — so `const PanLeft PanDirection = -1` reached the enum
// collector as an *ast.UnaryExpr and was dropped, yielding [0] instead of [-1, 0, 1].
//
// The explicit `+1` spelling exercises the other sign operator and must render without a stray plus.
func TestCoverage_Bug3412(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/3412/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	props := doc.Definitions["ControlParams"].Properties
	pan, ok := props["pan"]
	require.True(t, ok)

	assert.Equal(t, "integer", pan.Type[0])
	assert.Equal(t, []any{int64(-1), int64(0), int64(1)}, pan.Enum,
		"a negative const must not be dropped from the enum")
	assert.Equal(t,
		"-1 PanLeft pans to the left.\n0 NoPan does not pan.\n1 PanRight pans to the right.",
		pan.Extensions["x-go-enum-desc"],
		"the const→value mapping must carry the sign too")

	scantest.CompareOrDumpJSON(t, doc, "bugs_3412_schema.json")
}
