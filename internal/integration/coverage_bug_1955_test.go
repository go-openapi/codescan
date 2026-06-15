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

// TestCoverage_Bug1955 locks go-swagger issue #1955: a swagger:operation can use
// a swagger:parameters struct defined in a DIFFERENT package — they are matched
// by operation id across all scanned packages (the swagger:operation analog of
// #1742).
func TestCoverage_Bug1955(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1955/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/cross"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1, "the cross-package param binds to the swagger:operation")
	assert.Equal(t, "q", op.Parameters[0].Name)
	assert.Equal(t, "query", op.Parameters[0].In)
	// The YAML body responses attach alongside the cross-package params.
	_, ok := op.Responses.StatusCodeResponses[200]
	assert.True(t, ok)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1955_schema.json")
}
