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

// TestCoverage_ResponseHeaderRefLeak pins the Q2 fix on the
// response side: when a header field's Go type resolves through the
// schema builder's makeRef path (e.g. typed as a named struct),
// `responseTypable.SetRef` no-ops instead of writing `response.Schema.Ref`.
// The body-schema leak that used to land in the pre-fix emission
// is gone; the SimpleSchema exit validator catches the attempt via
// the probe's HasRef and emits CodeUnsupportedInSimpleSchema.
//
// Two variants pinned:
//
//  1. LeakResponse — TagHeader typed as named struct Tag, no
//     strfmt override. Post-fix: header stays empty (the named
//     struct can't be expressed as OAS v2 SimpleSchema); body
//     schema is NOT set; diagnostic fires.
//
//  2. LeakWithStrfmtResponse — same shape plus
//     `// swagger:strfmt uuid` on the field. Post-fix: header
//     gets {string, uuid} from the strfmt override (which runs
//     after the exit validator's reset); body schema is NOT
//     set; diagnostic still fires for the underlying ref attempt.
func TestCoverage_ResponseHeaderRefLeak(t *testing.T) {
	var got []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-header-ref-leak/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			got = append(got, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// Body-schema leak is gone for both responses.
	require.Contains(t, doc.Responses, "leakResponse")
	leak := doc.Responses["leakResponse"]
	assert.Nil(t, leak.Schema, "leakResponse.Schema must be nil — no body leak")
	require.Contains(t, leak.Headers, "X-Tag", "the header field must still surface as a header entry")

	require.Contains(t, doc.Responses, "leakWithStrfmtResponse")
	leakStrfmt := doc.Responses["leakWithStrfmtResponse"]
	assert.Nil(t, leakStrfmt.Schema, "leakWithStrfmtResponse.Schema must be nil — no body leak")
	require.Contains(t, leakStrfmt.Headers, "X-Tag-ID")
	xTagID := leakStrfmt.Headers["X-Tag-ID"]
	assert.Equal(t, "string", xTagID.Type, "strfmt override fires after the exit-validator reset")
	assert.Equal(t, "uuid", xTagID.Format)

	// SimpleSchema-mode diagnostic fires at least once.
	var seenForbiddenRef bool
	for _, d := range got {
		if d.Code == grammar.CodeUnsupportedInSimpleSchema &&
			strings.Contains(d.Message, "$ref") {
			seenForbiddenRef = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
			break
		}
	}
	if !seenForbiddenRef {
		for i, d := range got {
			t.Logf("diag[%d] code=%s severity=%s msg=%q", i, d.Code, d.Severity, d.Message)
		}
	}
	assert.True(t, seenForbiddenRef, "expected CodeUnsupportedInSimpleSchema diagnostic naming `$ref` for the header-as-named-struct attempt")

	// Golden pins the full shape (no body Schema on either response).
	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_header_ref_leak.json")
}
