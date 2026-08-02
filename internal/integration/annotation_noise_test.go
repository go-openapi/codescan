// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// A classifier annotation written where nothing consults it must be reported, not silently dropped.
//
// `swagger:strfmt` / `swagger:type` in an EMBEDDED field's own comment are parsed, validated and
// discarded, while the same annotation on a regular field one line away is honoured. That asymmetry
// is what makes the silence a defect rather than a rule — and because the scanner REJECTS an unknown
// annotation in that same comment (TestCoverage_UnknownAnnotation), the author got validation
// feedback implying the annotation was meaningful and nothing saying it had been dropped.
//
// The diagnostic reports the drop; it does not change it. Where such an annotation belongs is the
// embedded type's own declaration, and the message says so.
func TestAnnotationNoise(t *testing.T) {
	var diags []codescan.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/annotation-noise/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d codescan.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	find := func(code, needle string) (codescan.Diagnostic, bool) {
		for _, d := range diags {
			if string(d.Code) == code && strings.Contains(d.Message, needle) {
				return d, true
			}
		}

		return codescan.Diagnostic{}, false
	}

	t.Run("classifiers on an embed are reported as ineffective", func(t *testing.T) {
		d, ok := find("scan.ineffective-annotation", "swagger:strfmt and swagger:type")
		require.True(t, ok, "both annotations on one allOf embed must be reported together; got %v", diags)
		assert.Equal(t, codescan.SeverityWarning, d.Severity)
		assert.NotZero(t, d.Pos.Line, "the diagnostic must point at the offending embed")

		_, ok = find("scan.ineffective-annotation", "annotate the embedded type's own declaration")
		assert.True(t, ok, "the message must say where the annotation does belong")

		var n int
		for _, d := range diags {
			if string(d.Code) == "scan.ineffective-annotation" {
				n++
			}
		}
		assert.Equal(t, 2, n, "one report per annotated embed — the allOf one and the plain one")
	})

	t.Run("the annotations are still ignored, not applied", func(t *testing.T) {
		// The diagnostic reports the drop; it does not change it. An embed still contributes its
		// embedded type's shape, so the composed member is Target's object either way.
		host := doc.Definitions["IneffectiveOnAllOf"]
		require.Len(t, host.AllOf, 2)
		assert.Equal(t, "object{left}", schemaSignature(host.AllOf[0], doc.Definitions, 0))

		plain := doc.Definitions["IneffectiveOnPlain"]
		assert.Equal(t, "object{left,note}", schemaSignature(plain, doc.Definitions, 0))
	})

	t.Run("the same annotations on a regular field are honoured", func(t *testing.T) {
		// The control that makes the asymmetry a defect rather than a rule: identical syntax, identical
		// position in the comment, different outcome.
		props := doc.Definitions["EffectiveOnField"].Properties
		assert.Equal(t, "string/uuid", schemaSignature(props["fmt"], doc.Definitions, 0))
		assert.Equal(t, "string/", schemaSignature(props["typ"], doc.Definitions, 0))

		for _, d := range diags {
			assert.NotContains(t, d.Message, "EffectiveOnField",
				"a regular field must not be reported as ineffective")
		}
	})

	scantest.CompareOrDumpJSON(t, doc, "enhancements_annotation_noise.json")
}
