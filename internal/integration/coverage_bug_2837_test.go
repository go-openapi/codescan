// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"

	oaispec "github.com/go-openapi/spec"
)

// TestCoverage_Bug2837 locks the fix for go-swagger issue #2837 ("Responses defined in routes break
// with go 1.19 formatting"): gofmt rewrites a route's `+ name:` parameter bullet to `- name:`,
// which used to break parsing.
//
// Both the authored `+` form and the gofmt-canonical `-` form now parse to the same parameter.
func TestCoverage_Bug2837(t *testing.T) {
	scan := func(pkg string) *oaispec.Swagger {
		doc, err := codescan.Run(&codescan.Options{Packages: []string{pkg}, WorkDir: scantest.FixturesDir()})
		require.NoError(t, err)
		require.NotNil(t, doc)
		return doc
	}

	param := func(doc *oaispec.Swagger) oaispec.Parameter {
		pi, ok := doc.Paths.Paths["/foobar"]
		require.True(t, ok)
		require.NotNil(t, pi.Post)
		require.Len(t, pi.Post.Parameters, 1)
		return pi.Post.Parameters[0]
	}

	plus := param(scan("./bugs/2837a/..."))
	dash := param(scan("./bugs/2837b/..."))

	for _, p := range []oaispec.Parameter{plus, dash} {
		assert.Equal(t, "payload", p.Name)
		assert.Equal(t, "body", p.In)
		assert.True(t, p.Required)
		require.NotNil(t, p.Schema)
		assert.Equal(t, "string", p.Schema.Type[0])
	}
	assert.Equal(t, plus, dash, "the gofmt-canonical `-` form must parse identically to `+`")

	// Lock the gofmt-canonical (dash) form's full output.
	scantest.CompareOrDumpJSON(t, scan("./bugs/2837b/..."), "bugs_2837_schema.json")
}
