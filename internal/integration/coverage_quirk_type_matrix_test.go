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

// TestQuirk_TypeMatrix is the F3 WITNESS (documented-current): it captures how
// swagger:type renders today across its argument vocabulary, combined with
// swagger:strfmt, at both the field and named-type sites. No fix is applied —
// the golden quirk_type_matrix.json is the baseline the F3 rationalisation is
// measured against. Diagnostics (if any) are captured too.
//
// Read the golden to see: which args override vs fall through to the Go type
// ($ref Custom), whether integer/number/boolean are accepted, what []string /
// file / badValue / a scanned type name do, and whether strfmt-vs-type
// precedence differs between the field site and the named site.
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

	t.Logf("diagnostics emitted: %d", len(diags))
	for _, d := range diags {
		t.Logf("  %s", d)
	}

	matrix := doc.Definitions["Matrix"].Properties
	require.NotNil(t, matrix)

	// Baseline that is correct today and must stay correct after the fix.
	assert.Equal(t, []string{"string"}, []string(matrix["aString"].Type))
	assert.Equal(t, []string{"object"}, []string(matrix["fObject"].Type))
	// The `array` Mode-2 idiom on a Go slice — load-bearing, keep working.
	assert.Equal(t, []string{"array"}, []string(matrix["gArray"].Type))
	assert.Equal(t, []string{"string"}, []string(matrix["gArray"].Items.Schema.Type))

	// --- Contradictions the F3 reconciliation will flip (documented-current) ---

	// TODO F3: `integer`/`number`/`boolean` are grammar-valid but the builder
	// silently drops them — the field falls through to the inlined Go struct
	// instead of resolving to a scalar. After the fix these become scalars.
	for _, k := range []string{"bInteger", "dNumber", "eBoolean"} {
		assert.Equal(t, []string{"object"}, []string(matrix[k].Type),
			"TODO F3: %q is silently dropped today (should resolve to a scalar)", k)
	}

	// TODO F3: `int64` (a Go builtin) renders correctly, yet the grammar emits
	// a parse.invalid-type-ref ERROR for it — "errors but works". After the
	// fix the Go-builtin spelling must not raise that diagnostic.
	assert.Equal(t, []string{"integer"}, []string(matrix["cInt64"].Type))
	assert.Equal(t, "int64", matrix["cInt64"].Format)

	// TODO F3: []string is unsupported today (grammar error + builder drop).
	assert.Equal(t, []string{"object"}, []string(matrix["hArrayExplicit"].Type),
		"TODO F3: []string should become {type:array, items:{type:string}}")

	// TODO F3: file / a scanned-type name / a typo all fall through silently
	// (file should warn+ignore; unknowns should get one clean diagnostic).
	for _, k := range []string{"iFile", "jBad", "kScanned"} {
		assert.Equal(t, []string{"object"}, []string(matrix[k].Type),
			"TODO F3: %q falls through today", k)
	}

	// TODO F3: the grammar already errors on these four args (int64, []string,
	// badValue, Custom) — but with a vocabulary disjoint from the builder's.
	// The fix must reconcile the two layers into one consistent diagnostic.
	var invalidTypeRefs int
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidTypeRef {
			invalidTypeRefs++
		}
	}
	assert.Equal(t, 4, invalidTypeRefs,
		"TODO F3: grammar emits invalid-type-ref for int64/[]string/badValue/Custom today")

	scantest.CompareOrDumpJSON(t, doc, "quirk_type_matrix.json")
}
