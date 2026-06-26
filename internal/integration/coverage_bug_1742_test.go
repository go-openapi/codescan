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

// TestCoverage_Bug1742 locks go-swagger issue #1742 ("Parameters"): a swagger:parameters struct may
// live in a DIFFERENT package from the swagger:route that references it — as long as both
// packages are scanned, the parameter is bound to the operation by its operation id.
//
// This fixture also stands as the regression witness for go-swagger #1735: the legacy
// `scan/route_params.go` engine panicked ("assignment to entry in nil map") while parsing a route's
// params.
// That engine is gone; grammar2 scans a route-with-params (here, cross-package) without panicking.
func TestCoverage_Bug1742(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1742/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/things"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1, "the cross-package param struct is bound to the route")
	p := op.Parameters[0]
	assert.Equal(t, "limit", p.Name)
	assert.Equal(t, "query", p.In)
	assert.Equal(t, "integer", p.Type)
	assert.Equal(t, "Limit caps the number of results.", p.Description)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1742_schema.json")
}
