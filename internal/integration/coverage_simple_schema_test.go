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

// TestCoverage_SimpleSchemaViolation exercises M1's exit validator: a query parameter whose Go type
// resolves to an object-typed SimpleSchema fires CodeUnsupportedInSimpleSchema and the target is
// reset to empty `{}`.
//
// Plumbing tested:
//   - schema.WithSimpleSchema option carries the `in` value to the builder
//   - exit validator detects Type=="object" as a violation
//   - paramTypable.ResetForViolation wipes the SimpleSchema-shape
//   - OnDiagnostic callback fires with the new code
func TestCoverage_SimpleSchemaViolation(t *testing.T) {
	var got []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/simple-schema-violation/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			got = append(got, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// 1. Diagnostic surface.
	var seen bool
	for _, d := range got {
		if d.Code == grammar.CodeUnsupportedInSimpleSchema {
			seen = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity, "CodeUnsupportedInSimpleSchema severity")
			assert.Contains(t, d.Message, "SimpleSchema", "message should mention SimpleSchema")
			break
		}
	}
	assert.True(t, seen, "expected CodeUnsupportedInSimpleSchema diagnostic")

	// 2. Target reset. The offending parameter should have an empty
	//    SimpleSchema (no Type, no Format, no Ref) — honest over lossy.
	require.Contains(t, doc.Paths.Paths, "/violation")
	op := doc.Paths.Paths["/violation"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 1)
	bad := op.Parameters[0]
	assert.Equal(t, "bad", bad.Name)
	assert.Equal(t, "query", bad.In, "in: query preserved")
	assert.Empty(t, bad.Type, "Type should be wiped to empty")
	assert.Empty(t, bad.Format, "Format should be wiped to empty")
	assert.Empty(t, bad.Ref.String(), "Ref should be wiped to empty")
}
