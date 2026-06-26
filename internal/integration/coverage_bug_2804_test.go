// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2804 verifies the fix for go-swagger issue #2804: a map[string][]string field
// used to crash (parameters) or silently corrupt the output (response headers) — a Go map has no
// OAS2 SimpleSchema representation.
//
// The same guard now covers both SimpleSchema targets:
//
//   - a map swagger:parameters field (in=query) used to panic with a nil-ptr
//     dereference (parameterBuilder.buildFromField -> Schema.Typed on nil);
//   - a map response header field (in=header) used to write `object` onto the
//     response *body* schema and leave the header untyped.
//
// Fixed behaviour: each unrepresentable map field is guarded — the scan emits a located
// diagnostic (validate.unsupported-in-simple-schema) and skips the field instead of panicking or
// corrupting the output.
//
// This is also a concrete witness for the §8.1 scanner panic-recovery feature
// (forthcoming-features.md).
func TestCoverage_Bug2804(t *testing.T) {
	var diagnostics []grammar.Diagnostic

	spec, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2804/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	})

	// No panic, no error: the unrepresentable fields are guarded, not fatal.
	require.NoError(t, err)
	require.NotNil(t, spec)

	pathItem := spec.Paths.Paths["/q"]
	op := pathItem.Get
	require.NotNil(t, op)

	// The map parameter field is skipped: the operation carries no such param.
	for _, p := range op.Parameters {
		assert.NotEqual(t, "property_filters", p.Name,
			"the unrepresentable map param must be skipped, not emitted")
	}

	// The map response-header field is skipped, and the response body schema was NOT corrupted by the
	// map (the bug wrote `object` onto resp.Schema).
	resp := op.Responses.StatusCodeResponses[200]
	assert.NotContains(t, resp.Headers, "X-Filters",
		"the unrepresentable map header must be skipped, not emitted")
	assert.Nil(t, resp.Schema,
		"the map header must not leak `object` onto the response body schema")

	// A located diagnostic flags each skipped field.
	byMsg := map[string]grammar.Diagnostic{}
	for _, d := range diagnostics {
		if d.Code != grammar.CodeUnsupportedInSimpleSchema {
			continue
		}
		assert.Equal(t, grammar.SeverityWarning, d.Severity)
		assert.NotEmpty(t, d.Pos.Filename, "diagnostic must be located")
		switch {
		case strings.Contains(d.Message, "property_filters"):
			byMsg["param"] = d
		case strings.Contains(d.Message, "X-Filters"):
			byMsg["header"] = d
		}
	}
	assert.Contains(t, byMsg, "param",
		"expected a %s diagnostic for the unrepresentable map parameter", grammar.CodeUnsupportedInSimpleSchema)
	assert.Contains(t, byMsg, "header",
		"expected a %s diagnostic for the unrepresentable map response header", grammar.CodeUnsupportedInSimpleSchema)
}
