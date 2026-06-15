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

// TestQuirk_TypeMatrix verifies the F3 reconciliation: swagger:type is now a
// uniform "inline this type" directive (never a $ref), resolved consistently at
// the field site and the named-type site. It supersedes the documented-current
// witness this test started as. The golden quirk_type_matrix.json is the
// fixed baseline; this test pins the headline behaviours and the diagnostics.
//
// Vocabulary (always inlined): canonical OAS-2 scalars + Go-builtin spellings,
// `[]T` arrays, the `inline` keyword (expand the Go type), the deprecated
// `array` keyword, type-name references (inline a known definition), with a
// `file` / unknown diagnostic and a strfmt-iff-format-compatible rule.
func TestQuirk_TypeMatrix(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:     []string{"./quirks/swagger-type-matrix/..."},
		WorkDir:      scantest.FixturesDir(),
		ScanModels:   true,
		OnDiagnostic: func(d grammar.Diagnostic) { diags = append(diags, d) },
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	m := doc.Definitions["Matrix"].Properties
	require.NotNil(t, m)

	scalar := func(name, typ string) {
		t.Helper()
		assert.Equalf(t, []string{typ}, []string(m[name].Type), "%s → %s", name, typ)
	}

	// Canonical scalar names + Go builtins, all inlined.
	scalar("aString", "string")
	scalar("bInteger", "integer") // was silently dropped; now resolves
	scalar("dNumber", "number")
	scalar("eBoolean", "boolean")
	scalar("fObject", "object")
	scalar("cInt64", "integer")
	assert.Equal(t, "int64", m["cInt64"].Format) // Go builtin, no longer a grammar error

	// []T → array with inlined items.
	assert.Equal(t, []string{"array"}, []string(m["hArrayExplicit"].Type))
	assert.Equal(t, []string{"string"}, []string(m["hArrayExplicit"].Items.Schema.Type))

	// `array` keyword on a Go slice still inlines (deprecated; see diagnostics).
	assert.Equal(t, []string{"array"}, []string(m["gArray"].Type))
	assert.Equal(t, []string{"string"}, []string(m["gArray"].Items.Schema.Type))

	// `inline` expands the field's own Go type (Custom) in place — no $ref.
	nInline := m["nInline"]
	assert.Equal(t, []string{"object"}, []string(nInline.Type))
	assert.Contains(t, nInline.Properties, "x")
	assert.Empty(t, nInline.Ref.String())

	// Type-name reference inlines a known definition (no $ref), even across a
	// different Go type (pCrossRef is a Go string overridden to Custom).
	kScanned := m["kScanned"]
	assert.Contains(t, kScanned.Properties, "x")
	assert.Empty(t, kScanned.Ref.String())
	assert.Equal(t, []string{"object"}, []string(m["pCrossRef"].Type))
	assert.Contains(t, m["pCrossRef"].Properties, "x")

	// []Custom → array of the inlined Custom type.
	assert.Equal(t, []string{"array"}, []string(m["oArrayCustom"].Type))
	assert.Contains(t, m["oArrayCustom"].Items.Schema.Properties, "x")

	// strfmt + type precedence: type wins; a compatible format is applied, an
	// incompatible one is dropped (with a diagnostic, asserted below).
	assert.Equal(t, []string{"string"}, []string(m["mStrfmtThenString"].Type))
	assert.Equal(t, "uuid", m["mStrfmtThenString"].Format) // uuid compatible with string
	assert.Equal(t, []string{"integer"}, []string(m["lStrfmtThenType"].Type))
	assert.Empty(t, m["lStrfmtThenType"].Format) // uuid incompatible with integer → dropped

	// file / unknown fall through to the Go type (inlined), with a diagnostic.
	assert.Contains(t, m["iFile"].Properties, "x")
	assert.Contains(t, m["jBad"].Properties, "x")

	// Named-type site behaves identically to the field site.
	assert.Equal(t, []string{"string"}, []string(doc.Definitions["NamedScalarString"].Type))
	assert.Equal(t, []string{"array"}, []string(doc.Definitions["NamedSlice"].Type))
	assert.Equal(t, []string{"integer"}, []string(doc.Definitions["NamedStrfmtAndType"].Type))

	// Diagnostics: array-deprecated ×2 (field gArray + named NamedSlice),
	// strfmt-incompatible ×2 (field lStrfmtThenType + named NamedStrfmtAndType),
	// file ×1, unknown ×1. No grammar invalid-type-ref any more.
	byCode := map[grammar.Code]int{}
	for _, d := range diags {
		byCode[d.Code]++
		assert.NotEqualf(t, grammar.CodeInvalidTypeRef, d.Code, "no grammar invalid-type-ref: %s", d)
	}
	assert.Equal(t, 2, byCode[grammar.CodeDeprecated], "array deprecation, field + named")
	assert.Equal(t, 2, byCode[grammar.CodeShapeMismatch], "strfmt incompatible, field + named")
	assert.Equal(t, 2, byCode[grammar.CodeUnsupportedType], "file + unknown")

	scantest.CompareOrDumpJSON(t, doc, "quirk_type_matrix.json")
}
