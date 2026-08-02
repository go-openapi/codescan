// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_SimpleSchemaViolation covers the two ways a non-body parameter can fail to be an
// OAS v2 SimpleSchema. They look alike and have different remedies, so both are pinned here.
//
//  1. The ANNOTATION asks for something the location cannot carry (`swagger:type object`). Since
//     `type` is mandatory under SimpleSchema, the override is refused before it is applied and the
//     Go-derived type stands, leaving a valid parameter. The diagnostic names the annotation.
//     This case used to be honoured and then wiped, which produced an untyped parameter.
//  2. The GO TYPE itself is not representable (a struct). Nothing to refuse, nothing to fall back
//     to, so the exit validator wipes the target — honest over lossy.
//
// Plumbing tested:
//   - schema.WithSimpleSchema option carries the `in` value to the builder
//   - the override gate refuses an object-resolving swagger:type under SimpleSchema
//   - exit validator detects Type=="object" as a violation
//   - paramTypable.ResetForViolation wipes the SimpleSchema-shape
//   - OnDiagnostic callback fires with the code in both cases
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

	require.Contains(t, doc.Paths.Paths, "/violation")
	op := doc.Paths.Paths["/violation"].Get
	require.NotNil(t, op)
	require.Len(t, op.Parameters, 2)

	byName := make(map[string]oaispec.Parameter, len(op.Parameters))
	for _, p := range op.Parameters {
		byName[p.Name] = p
	}

	// Case 1 — the override is refused, the Go type stands. A parameter without a type is not a
	// valid SimpleSchema, so keeping `string` is the whole point of refusing.
	bad, ok := byName["bad"]
	require.True(t, ok, "missing parameter bad")
	assert.Equal(t, "query", bad.In, "in: query preserved")
	assert.Equal(t, "string", bad.Type, "the Go-derived type must survive a refused override")
	assert.Empty(t, bad.Ref.String(), "Ref is forbidden under SimpleSchema")

	// Case 2 — nothing to fall back to, so the target is wiped.
	unrep, ok := byName["unrepresentable"]
	require.True(t, ok, "missing parameter unrepresentable")
	assert.Empty(t, unrep.Type, "an unrepresentable Go type is wiped")
	assert.Empty(t, unrep.Format, "Format wiped with it")
	assert.Empty(t, unrep.Ref.String(), "Ref wiped with it")
}
