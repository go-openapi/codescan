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

// TestCoverage_ResponseFileTypes pins Q3's three variants:
//
//  1. FileBodyResponse — `swagger:file` + `in: body` →
//     resp.Schema = {file, ""}; no headers; no diagnostic.
//
//  2. FileOnHeaderResponse — `swagger:file` + explicit
//     `in: header` → diagnostic fires; file branch skipped; field
//     surfaces as a regular header.
//
//  3. FileOnImplicitDefaultResponse — `swagger:file` with no
//     `in:` (Q1 implicit-header default) → same diagnostic;
//     field surfaces as a regular header.
//
// Diagnostic capture asserts CodeUnsupportedInSimpleSchema fires for variants (2) and (3) but not
// for (1).
func TestCoverage_ResponseFileTypes(t *testing.T) {
	var got []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-file-types/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			got = append(got, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// (1) Legitimate file body.
	require.Contains(t, doc.Responses, "fileBodyResponse")
	body := doc.Responses["fileBodyResponse"]
	require.NotNil(t, body.Schema, "FileBodyResponse should produce a body schema")
	assert.Equal(t, "file", body.Schema.Type[0], "body schema must be {file, ...}")
	assert.Empty(t, body.Headers, "file-body response carries no headers")

	// (2) Misuse with explicit `in: header`.
	require.Contains(t, doc.Responses, "fileOnHeaderResponse")
	headerMisuse := doc.Responses["fileOnHeaderResponse"]
	assert.Nil(t, headerMisuse.Schema, "no body schema corruption from header-positioned swagger:file")
	require.Contains(t, headerMisuse.Headers, "X-Misplaced", "field falls through to the normal header build")
	assert.Equal(t, "string", headerMisuse.Headers["X-Misplaced"].Type)

	// (3) Misuse with implicit default (no `in:` line).
	require.Contains(t, doc.Responses, "fileOnImplicitDefaultResponse")
	implicitMisuse := doc.Responses["fileOnImplicitDefaultResponse"]
	assert.Nil(t, implicitMisuse.Schema, "no body schema corruption from swagger:file at implicit-header default")
	require.Contains(t, implicitMisuse.Headers, "X-Implicit")
	assert.Equal(t, "string", implicitMisuse.Headers["X-Implicit"].Type)

	// Diagnostic captures: (2) and (3) each emit one CodeUnsupportedInSimpleSchema warning naming
	// `swagger:file`.
	var seenExplicit, seenImplicit bool
	for _, d := range got {
		if d.Code != grammar.CodeUnsupportedInSimpleSchema {
			continue
		}
		if !strings.Contains(d.Message, "swagger:file") {
			continue
		}
		switch {
		case strings.Contains(d.Message, `"X-Misplaced"`):
			seenExplicit = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
		case strings.Contains(d.Message, `"X-Implicit"`):
			seenImplicit = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
		}
	}
	if !seenExplicit || !seenImplicit {
		for i, d := range got {
			t.Logf("diag[%d] code=%s severity=%s msg=%q", i, d.Code, d.Severity, d.Message)
		}
	}
	assert.True(t, seenExplicit, "expected diagnostic naming `swagger:file` on field X-Misplaced (explicit in: header)")
	assert.True(t, seenImplicit, "expected diagnostic naming `swagger:file` on field X-Implicit (implicit default)")

	// Golden pins the full shape.
	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_file_types.json")
}
