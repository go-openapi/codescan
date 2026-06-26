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

// TestDiagnostics_UnsupportedGoType locks the Warning that replaced the legacy stderr "unsupported
// Go type" log line.
//
// A struct field typed complex128 has no Swagger 2.0 representation: its schema is left untyped
// (the build skips it) and a validate.unsupported-go-type Warning is delivered through OnDiagnostic
// (never to stderr).
func TestDiagnostics_UnsupportedGoType(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./enhancements/unsupported-go-type/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The representable field is typed; the unrepresentable one is left untyped (the builder skipped
	// it after warning rather than guessing a type).
	widget, ok := doc.Definitions["Widget"]
	require.True(t, ok, "Widget must be discovered")
	require.Contains(t, widget.Properties, "name")
	assert.Equal(t, "string", widget.Properties["name"].Type[0])
	require.Contains(t, widget.Properties, "weird")
	assert.Empty(t, widget.Properties["weird"].Type, "complex128 has no Swagger type")

	// A Warning diagnostic names the offending type.
	//
	// The position falls back to the declaration (no finer node is threaded into the type dispatch).
	var found bool
	for _, d := range diags {
		if d.Code == grammar.CodeUnsupportedGoType {
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
			assert.Contains(t, d.Message, "complex128")
			found = true
		}
	}
	assert.True(t, found, "a validate.unsupported-go-type Warning must be emitted")
}

// TestDiagnostics_IgnoredByTag locks the Hint that replaced the legacy debug log line for
// tag-filtered routes.
//
// Excluding the "orders" tag drops the petstore's order routes and surfaces a scan.ignored-by-tag
// Hint per dropped route through OnDiagnostic.
func TestDiagnostics_IgnoredByTag(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./goparsing/petstore/..."},
		WorkDir:      scantest.FixturesDir(),
		ExcludeTags:  []string{"orders"},
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// The excluded routes are absent from the spec.
	for path := range doc.Paths.Paths {
		assert.NotContains(t, path, "/orders", "order routes must be excluded")
	}

	// Hints (not Warnings/Errors) were emitted for the dropped routes.
	var hints int
	for _, d := range diags {
		if d.Code == grammar.CodeIgnoredByTag {
			assert.Equal(t, grammar.SeverityHint, d.Severity)
			assert.Contains(t, d.Message, "tag rules")
			hints++
		}
	}
	assert.Positive(t, hints, "at least one scan.ignored-by-tag Hint must be emitted")
}
