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

// TestCoverage_Bug2575 locks go-swagger issue #2575 ("custom request headers"): a
// swagger:parameters field marked `in: header` emits a header parameter.
func TestCoverage_Bug2575(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2575/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/pets"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	p := op.Parameters[0]
	assert.Equal(t, "header", p.In)
	assert.Equal(t, "X-Request-Id", p.Name)
	assert.Equal(t, "string", p.Type)
	assert.True(t, p.Required)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2575_schema.json")
}
