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
// The quoted source example (`example: "123456"`) emits the quote-stripped
// value `123456`: surrounding quotes are delimiters (quirk F8, fixed alongside
// go-swagger#2547).
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
	// F8 (fixed): surrounding quotes are stripped from the string example.
	assert.Equal(t, "123456", p.Schema.Example, "surrounding quotes on a string example are stripped (F8)")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2899_schema.json")
}
