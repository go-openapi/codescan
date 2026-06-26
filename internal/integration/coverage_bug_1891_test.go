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

// TestCoverage_Bug1891 locks go-swagger issue #1891 ("group type"): a model declared inside a
// grouped `type ( ... )` block, plus a swagger:route declared inside a function body, are both
// discovered and produce a valid spec.
func TestCoverage_Bug1891(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/1891/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	// Model from the grouped type declaration.
	foo, ok := doc.Definitions["FooOutput"]
	require.True(t, ok, "a swagger:model inside a type(...) group must be discovered")
	assert.Contains(t, foo.Properties, "addr")

	// Route declared inside a function body, with query params and a body ref.
	op := doc.Paths.Paths["/foo"].Get
	require.NotNil(t, op, "a swagger:route inside a func body must be discovered")
	assert.Equal(t, "fooInput", op.ID)
	assert.Len(t, op.Parameters, 2)
	r200 := op.Responses.StatusCodeResponses[200]
	require.NotNil(t, r200.Schema)
	assert.Equal(t, "#/definitions/FooOutput", r200.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_1891_schema.json")
}
