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

// TestCoverage_Bug2251 asserts the EXPECTED resolution of go-swagger issue #2251 ("map with
// non-string keys").
//
// Currently RED for integer-kind keys.
//
// encoding/json marshals integer-kind map keys (map[int]V, map[int64]V) as JSON string keys, so
// they are representable as {type:object, additionalProperties:V} — but buildFromMap only emits
// additionalProperties for string / TextMarshaler keys and silently drops the rest.
//
// The string-key form is the green guard rail. (Part of the additionalProperties work on this
// branch; see forthcoming §18.)
func TestCoverage_Bug2251(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2251/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)

	props := doc.Definitions["M"].Properties
	hasAP := func(name string) bool {
		p, ok := props[name]
		return ok && p.AdditionalProperties != nil && p.AdditionalProperties.Schema != nil
	}

	// Guard rail: string keys already emit additionalProperties.
	assert.True(t, hasAP("strKey"), "string-keyed map emits additionalProperties")

	// The #2251 defect: integer-kind keys must emit additionalProperties too.
	assert.True(t, hasAP("intKey"), "map[int]V must emit additionalProperties (go-swagger#2251)")
	assert.True(t, hasAP("i64Key"), "map[int64]V must emit additionalProperties (go-swagger#2251)")

	// Fail-loud half (§18): a json-illegal key (float) emits no additionalProperties AND raises a
	// CodeUnsupportedType diagnostic instead of dropping silently.
	assert.False(t, hasAP("badKey"), "map[float64]V is not json-representable; no additionalProperties")

	var unsupported []grammar.Diagnostic
	for _, d := range diags {
		if d.Code == grammar.CodeUnsupportedType {
			unsupported = append(unsupported, d)
		}
	}
	require.Len(t, unsupported, 1, "exactly one unsupported-key diagnostic")
	assert.Equal(t, grammar.SeverityWarning, unsupported[0].Severity)
	assert.Contains(t, unsupported[0].Message, "float64")
	assert.Contains(t, unsupported[0].Message, "additionalProperties dropped")
}
