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

// TestCoverage_Bug2245 locks go-swagger issue #2245 ("swagger:response that produces
// application/xml"): a `produces: [application/xml]` in the swagger:operation YAML body sets the
// operation's produced media type.
func TestCoverage_Bug2245(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2245/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/xml"].Get
	require.NotNil(t, op)
	assert.Equal(t, []string{"application/xml"}, op.Produces)

	scantest.CompareOrDumpJSON(t, doc, "bugs_2245_schema.json")
}
