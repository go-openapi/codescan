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

// TestCoverage_Bug2353 locks go-swagger issue #2353 ("valid spec with body and
// request param"): an operation that mixes a path parameter and a body parameter
// produces a valid parameter set — the body param is a clean {in:body, schema:
// {$ref}} with no forbidden sibling `type`, and the path param is a simple
// {in:path, type:string}.
func TestCoverage_Bug2353(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2353/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	op := doc.Paths.Paths["/test/{id}"].Post
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 2)
	var path, body int = -1, -1
	for i, p := range op.Parameters {
		switch p.In {
		case "path":
			path = i
		case "body":
			body = i
		}
	}
	require.GreaterOrEqual(t, path, 0)
	require.GreaterOrEqual(t, body, 0)
	assert.Equal(t, "string", op.Parameters[path].Type)
	bp := op.Parameters[body]
	assert.Empty(t, bp.Type, "a body param must not carry a forbidden sibling type (go-swagger#2353)")
	require.NotNil(t, bp.Schema)
	assert.Equal(t, "#/definitions/Req", bp.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2353_schema.json")
}
