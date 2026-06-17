// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/parsers/grammar"
	"github.com/go-openapi/codescan/internal/scantest"
	oaispec "github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_PrimitiveResponse locks the primitive response-body matrix:
// tagged primitive (body:integer), array-of-primitive (body:[]string), the
// swagger:response array-body workaround, and a model-ref control that must
// still resolve to a $ref.
func TestCoverage_PrimitiveResponse(t *testing.T) {
	var diags []grammar.Diagnostic
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./enhancements/primitive-response/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
		OnDiagnostic: func(d grammar.Diagnostic) {
			diags = append(diags, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	resp200 := func(path string) oaispec.Response {
		op := doc.Paths.Paths[path].Get
		require.NotNil(t, op, path)
		require.NotNil(t, op.Responses, path)
		r, ok := op.Responses.StatusCodeResponses[200]
		require.True(t, ok, path)
		return r
	}

	// body:integer → {type: integer}
	tagged := resp200("/tagged")
	require.NotNil(t, tagged.Schema)
	assert.Equal(t, []string{"integer"}, []string(tagged.Schema.Type))

	// body:[]string → {type: array, items: {type: string}}
	arr := resp200("/array")
	require.NotNil(t, arr.Schema)
	assert.Equal(t, []string{"array"}, []string(arr.Schema.Type))
	require.NotNil(t, arr.Schema.Items)
	require.NotNil(t, arr.Schema.Items.Schema)
	assert.Equal(t, []string{"string"}, []string(arr.Schema.Items.Schema.Type))

	// body:Thing (model control) → $ref to the definition, NOT a primitive.
	thing := resp200("/thing")
	require.NotNil(t, thing.Schema)
	assert.Equal(t, "#/definitions/Thing", thing.Schema.Ref.String())
	assert.Empty(t, thing.Schema.Type)

	// 200: arrayBodyResponse → $ref to the named response, which itself
	// carries an array-of-primitive body schema.
	named := resp200("/named")
	assert.Equal(t, "#/responses/arrayBodyResponse", named.Ref.String())
	nr, ok := doc.Responses["arrayBodyResponse"]
	require.True(t, ok)
	require.NotNil(t, nr.Schema)
	assert.Equal(t, []string{"array"}, []string(nr.Schema.Type))
	require.NotNil(t, nr.Schema.Items)
	require.NotNil(t, nr.Schema.Items.Schema)
	assert.Equal(t, []string{"string"}, []string(nr.Schema.Items.Schema.Type))

	// Bare/untagged primitive (`200: string`) is NOT supported: the token
	// is read as a response name, the response is dropped, and a diagnostic
	// points the author at the unambiguous `body:` form.
	bare := doc.Paths.Paths["/bare"].Get
	require.NotNil(t, bare)
	if bare.Responses != nil {
		_, has := bare.Responses.StatusCodeResponses[200]
		assert.False(t, has, "bare `200: string` should be dropped, not promoted to a schema")
	}
	var hinted bool
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidAnnotation && strings.Contains(d.Message, "body:string") {
			hinted = true
		}
	}
	assert.True(t, hinted, "expected a diagnostic pointing at `body:string`")

	// Reserved type keyword after body: (`body:file`) is not a valid response
	// body type: dropped with a diagnostic. array/object/null behave the same.
	reserved := doc.Paths.Paths["/reserved"].Get
	require.NotNil(t, reserved)
	if reserved.Responses != nil {
		_, has := reserved.Responses.StatusCodeResponses[200]
		assert.False(t, has, "`body:file` should be dropped, not emitted")
	}
	var reservedDiag bool
	for _, d := range diags {
		if d.Code == grammar.CodeInvalidAnnotation && strings.Contains(d.Message, "not a supported response body type") {
			reservedDiag = true
		}
	}
	assert.True(t, reservedDiag, "expected a diagnostic rejecting `body:file`")

	scantest.CompareOrDumpJSON(t, doc, "primitive_response.json")
}
