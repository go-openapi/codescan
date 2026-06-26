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

// TestCoverage_Bug2020 locks go-swagger issue #2020 ("parameters not detected with multiple structs
// in one statement"): two swagger:parameters structs declared in a single grouped `type ( ... )`
// block are each detected and bound to their operation by id.
//
// (The reporter's exotic anonymous-inline-body field marked `swagger:name -` remains mis-handled
// — it is dropped to a query param — but that narrow edge is independent of the
// grouped-declaration detection this locks.)
func TestCoverage_Bug2020(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2020/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	alpha := doc.Paths.Paths["/alpha"].Get
	require.NotNil(t, alpha)
	require.Len(t, alpha.Parameters, 1, "param struct from the grouped block binds to alphaOp")
	assert.Equal(t, "q", alpha.Parameters[0].Name)

	beta := doc.Paths.Paths["/beta"].Get
	require.NotNil(t, beta)
	require.Len(t, beta.Parameters, 1, "the second grouped param struct binds to betaOp")
	assert.Equal(t, "r", beta.Parameters[0].Name)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2020_schema.json")
}
