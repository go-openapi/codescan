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

// TestCoverage_ResponseImplicitHeader exercises the four Q1 variants on the response side:
//
//  1. EmptyResponse — empty struct under swagger:response →
//     description-only response, no Headers map, no Schema.
//  2. AllHeadersResponse — fields default to header (Etag with no
//     `in:` line; RateLimit with explicit `in: header`). The body
//     schema stays nil.
//  3. MixedResponse — Tag (implicit header), Limit (explicit
//     header), Body (explicit `in: body`).
//  4. InvalidInResponse — `in: cookie` emits a
//     CodeInvalidAnnotation warning and defaults to header anyway
//     (diagnosed but not silently ignored).
//
// The full document is golden-captured so the shape of each response — Headers map presence,
// Schema presence, primitive types — locks down side-by-side.
func TestCoverage_ResponseImplicitHeader(t *testing.T) {
	var got []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/response-implicit-header/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			got = append(got, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// --- Variant 1: EmptyResponse ---
	require.Contains(t, doc.Responses, "emptyResponse")
	empty := doc.Responses["emptyResponse"]
	assert.Empty(t, empty.Headers, "empty struct should not emit a Headers map")
	assert.Nil(t, empty.Schema, "empty struct should not emit a body Schema")

	// --- Variant 2: AllHeadersResponse ---
	require.Contains(t, doc.Responses, "allHeadersResponse")
	allHeaders := doc.Responses["allHeadersResponse"]
	assert.Contains(t, allHeaders.Headers, "ETag", "Etag (no `in:`) should default to a header")
	assert.Contains(t, allHeaders.Headers, "X-Rate-Limit", "RateLimit (explicit `in: header`) should be a header")
	assert.Nil(t, allHeaders.Schema, "no `in: body` field → no body Schema")
	assert.Equal(t, "string", allHeaders.Headers["ETag"].Type)
	assert.Equal(t, "integer", allHeaders.Headers["X-Rate-Limit"].Type)

	// --- Variant 3: MixedResponse ---
	require.Contains(t, doc.Responses, "mixedResponse")
	mixed := doc.Responses["mixedResponse"]
	assert.Contains(t, mixed.Headers, "X-Tag", "Tag (implicit) should be a header")
	assert.Contains(t, mixed.Headers, "X-Limit", "Limit (explicit) should be a header")
	assert.NotContains(t, mixed.Headers, "body", "Body (explicit `in: body`) should not appear in Headers")
	require.NotNil(t, mixed.Schema, "Body field should populate resp.Schema")

	// --- Variant 4: InvalidInResponse ---
	require.Contains(t, doc.Responses, "invalidInResponse")
	invalidResp := doc.Responses["invalidInResponse"]
	assert.Contains(t, invalidResp.Headers, "Cookie", "`in: cookie` → still treated as header (Q1: not silently ignored)")

	var seenInvalid bool
	for _, d := range got {
		if d.Code == grammar.CodeInvalidAnnotation &&
			strings.Contains(d.Message, "in: cookie") &&
			strings.Contains(d.Message, "Cookie") {
			seenInvalid = true
			assert.Equal(t, grammar.SeverityWarning, d.Severity)
			break
		}
	}
	if !seenInvalid {
		for i, d := range got {
			t.Logf("diag[%d] code=%s severity=%s msg=%q", i, d.Code, d.Severity, d.Message)
		}
	}
	assert.True(t, seenInvalid, "expected a CodeInvalidAnnotation diagnostic naming `in: cookie` on field Cookie")

	// Golden of the full doc — shape-level pin (Headers maps, Schema presence, primitive types per
	// response).
	scantest.CompareOrDumpJSON(t, doc, "enhancements_response_implicit_header.json")
}
