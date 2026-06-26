// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2909 verifies the fix for go-swagger issue #2909: a route path with an inline
// regex segment ({id:[0-9]+}) was silently dropped because rxPath's alphabet lacks `[`/`]`.
//
// The fix strips the regex constraint to the RFC 6570 URI Template Level-1 form ({id}) before
// matching, so the route is emitted as /items/{id}, and warns that the constraint was dropped
// (OpenAPI 2.0 path templating cannot express it).
func TestCoverage_Bug2909(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2909/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	// FIXED: the regex segment is stripped to /items/{id} and the route emits.
	require.Contains(t, doc.Paths.Paths, "/items/{id}")
	assert.NotContains(t, doc.Paths.Paths, "/items/{id:[0-9]+}")
	assert.NotNil(t, doc.Paths.Paths["/items/{id}"].Get)

	// FIXED: a warning surfaces the dropped constraint (RFC 6570 Level-1 only).
	var warned bool
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidAnnotation && d.Severity == grammar.SeverityWarning {
			assert.Contains(t, d.Message, "id")
			assert.Contains(t, d.Message, "RFC 6570")
			warned = true
		}
	}
	assert.True(t, warned, "expected a stripped-regex warning")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2909_schema.json")
}
