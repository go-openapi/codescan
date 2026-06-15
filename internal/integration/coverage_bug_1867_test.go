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

// TestCoverage_Bug1867 locks go-swagger issue #1867 ("PATCH operation"): PATCH
// works through BOTH swagger:route (with a body parameter) and swagger:operation
// — the reporter found swagger:operation silently failed and swagger:route could
// not specify the JSON body.
func TestCoverage_Bug1867(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1867/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)

	// swagger:route PATCH with a body param.
	route := doc.Paths.Paths["/hybrid/route"].Patch
	require.NotNil(t, route, "swagger:route PATCH must emit a patch operation")
	require.Len(t, route.Parameters, 1)
	assert.Equal(t, "body", route.Parameters[0].In)
	require.NotNil(t, route.Parameters[0].Schema)
	assert.Contains(t, route.Parameters[0].Schema.Properties, "end_time")

	// swagger:operation PATCH.
	op := doc.Paths.Paths["/hybrid/op"].Patch
	require.NotNil(t, op, "swagger:operation PATCH must emit a patch operation")
	assert.Equal(t, "patchOpenHybridOp", op.ID)

	scantest.CompareOrDumpJSON(t, doc, "bugs_1867_schema.json")
}
