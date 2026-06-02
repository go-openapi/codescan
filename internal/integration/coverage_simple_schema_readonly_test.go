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

// TestCoverage_SimpleSchemaReadOnlyGate pins the schema builder's
// SimpleSchema-mode gate on the full-Schema-only `readOnly:`
// keyword. When the schema builder is invoked via WithSimpleSchema
// and walks an anonymous struct that carries a `readOnly: true`
// sub-field annotation, the Bool handler emits a
// CodeUnsupportedInSimpleSchema diagnostic naming `readOnly` and
// skips the write.
//
// The exit validator does NOT additionally reset the parameter here.
// The struct-walking dance writes through a throwaway scratch schema
// (paramTypable.Schema() returns nil for non-body), so the
// parameter's SimpleSchema stays at Type="" — the validator's "any"
// branch accepts that. The observable signal is the gate diagnostic
// itself.
func TestCoverage_SimpleSchemaReadOnlyGate(t *testing.T) {
	var got []grammar.Diagnostic
	_, err := codescan.Run(&codescan.Options{
		Packages: []string{"./enhancements/simple-schema-readonly/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			got = append(got, d)
		},
	})
	require.NoError(t, err)

	var seen bool
	for _, d := range got {
		if d.Code != grammar.CodeUnsupportedInSimpleSchema {
			continue
		}
		if strings.Contains(d.Message, "readOnly") {
			seen = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
			assert.Contains(t, d.Message, "full-Schema-only")
			break
		}
	}
	if !seen {
		for i, d := range got {
			t.Logf("diag[%d] code=%s severity=%s msg=%q", i, d.Code, d.Severity, d.Message)
		}
	}
	assert.True(t, seen, "expected a CodeUnsupportedInSimpleSchema diagnostic naming `readOnly`")
}
