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

// TestCoverage_Bug2899 locks the fix for go-swagger issue #2899 ("Example not
// being added to schema for body string params"): an `example:` on an inline
// body parameter of primitive type is now carried onto the parameter's schema.
//
// Known residual — quirk F8 (doc-site-quirks.md): a quoted string example
// (`example: "123456"`) keeps its source quotes, so the emitted value is
// `"123456"` rather than `123456`. Asserted here as-is; when F8 lands (strip
// surrounding quotes for string-typed example/default), update this assertion.
func TestCoverage_Bug2899(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2899/..."},
		WorkDir:  scantest.FixturesDir(),
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	pi, ok := doc.Paths.Paths["/login"]
	require.True(t, ok)
	require.NotNil(t, pi.Post)
	require.Len(t, pi.Post.Parameters, 1)

	p := pi.Post.Parameters[0]
	assert.Equal(t, "body", p.In)
	require.NotNil(t, p.Schema)
	assert.Equal(t, "string", p.Schema.Type[0])
	require.NotNil(t, p.Schema.Example, "the example must be carried onto the body param schema (the #2899 fix)")
	// F8 residual: the surrounding quotes are currently retained.
	assert.Equal(t, `"123456"`, p.Schema.Example, "TODO F8: should strip surrounding quotes -> 123456")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2899_schema.json")
}
