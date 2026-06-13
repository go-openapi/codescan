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

// TestCoverage_PathRegexStripping locks inline-regex path-parameter stripping
// across both the swagger:route and swagger:operation builders, plus the
// no-op control (a bare {id} template raises no warning).
func TestCoverage_PathRegexStripping(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/path-regex-stripping/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)
	require.NotNil(t, doc.Paths)

	// Multi-param route: both segments stripped.
	assert.Contains(t, doc.Paths.Paths, "/a/{x}/b/{y}")
	// Operation with a nested-brace quantifier: stripped to {code}.
	assert.Contains(t, doc.Paths.Paths, "/codes/{code}")
	// Control: plain template emitted unchanged.
	assert.Contains(t, doc.Paths.Paths, "/plain/{id}")

	// Warnings: one per route/operation that carried a constraint (the
	// multi-param route reports both names in a single diagnostic); the
	// plain control raises none.
	var warns int
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidAnnotation && d.Severity == grammar.SeverityWarning {
			warns++
			assert.Contains(t, d.Message, "RFC 6570")
		}
	}
	assert.Equal(t, 2, warns, "one warning for the multi-param route, one for the operation; none for the control")

	scantest.CompareOrDumpJSON(t, doc, "path_regex_stripping.json")
}
