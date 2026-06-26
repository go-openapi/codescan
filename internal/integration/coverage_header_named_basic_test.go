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

// TestCoverage_HeaderNamedBasic pins the M1-follow-up fix that brought `in: header` into the
// SimpleSchema-aware primitive-inline arm of classifierNamedBasic.
//
// Pre-fix, a header parameter typed as a named string emitted a `$ref` (invalid under OAS v2
// SimpleSchema); post-fix it inlines as `{type: string}`.
func TestCoverage_HeaderNamedBasic(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/header-named-basic/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	require.Contains(t, doc.Paths.Paths, "/header-named-basic")
	op := doc.Paths.Paths["/header-named-basic"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)

	session := op.Parameters[0]
	assert.Equal(t, "X-Session", session.Name)
	assert.Equal(t, "header", session.In)
	assert.Equal(t, "string", session.Type, "named basic should inline as primitive under SimpleSchema")
	assert.Empty(t, session.Ref.String(), "no $ref should be emitted")

	// The SessionID type should NOT appear as a top-level definition — the primitive-inline arm
	// bypasses FindModel.
	_, defined := doc.Definitions["SessionID"]
	assert.False(t, defined, "SessionID should not become a top-level definition")
}
