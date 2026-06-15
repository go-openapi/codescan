// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2651 locks the fix for go-swagger issue #2651: an operation
// mixing inline `swagger:operation` parameters with a `swagger:parameters`-bound
// struct used to mis-bind — the bound body's `$ref` schema was welded onto the
// inline `id` PATH parameter (invalid: a path param cannot carry a body schema),
// instead of being emitted as a separate body parameter.
//
// After the fix the bound body is its own `in: body` parameter and the inline
// path param carries only its simple schema (no Schema/$ref).
func TestCoverage_Bug2651(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2651/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pi := doc.Paths.Paths["/v1/users/{id}"]
	require.NotNil(t, pi.Patch)
	require.Len(t, pi.Patch.Parameters, 2, "path id + bound body are distinct parameters (go-swagger#2651)")

	byIn := map[string]oaispec.Parameter{}
	for _, p := range pi.Patch.Parameters {
		byIn[p.In] = p
	}

	path, ok := byIn["path"]
	require.True(t, ok, "the inline path parameter must be present")
	assert.Equal(t, "id", path.Name)
	assert.True(t, path.Required)
	assert.Nil(t, path.Schema, "a path param must not carry a body schema (go-swagger#2651)")

	body, ok := byIn["body"]
	require.True(t, ok, "the bound body parameter must be emitted separately")
	require.NotNil(t, body.Schema)
	assert.Equal(t, "#/definitions/UpdateUser", body.Schema.Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2651_schema.json")
}
