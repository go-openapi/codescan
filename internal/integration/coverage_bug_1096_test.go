// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

//go:build cgo

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug1096 locks the fix for go-swagger issue #1096 ("annotated go
// code fails to generate spec when using cgo"): a package that imports "C" used
// to yield an empty spec; it now scans normally via go/packages. Guarded by the
// cgo build tag so it is skipped where cgo is unavailable (the fixture is
// likewise cgo-tagged).
func TestCoverage_Bug1096(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1096/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc.Paths)

	op := doc.Paths.Paths["/orders"].Post
	require.NotNil(t, op, "the route is discovered despite cgo")
	assert.Equal(t, "createOrder", op.ID)
	require.Contains(t, doc.Responses, "order")

	scantest.CompareOrDumpJSON(t, doc, "bugs_1096_schema.json")
}
