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

// TestCoverage_Bug2218 locks go-swagger issue #2218 ("unable to connect a parameter to a route"): a
// swagger:parameters struct is bound to its route when the struct's operation id matches the
// route's operation id — the query parameter appears on the operation.
func TestCoverage_Bug2218(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2218/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1, "the param struct binds by matching operation id")
	assert.Equal(t, "filter", op.Parameters[0].Name)
	assert.Equal(t, "query", op.Parameters[0].In)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2218_schema.json")
}
