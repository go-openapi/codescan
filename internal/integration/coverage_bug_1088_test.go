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

// TestCoverage_Bug1088 locks the fix for go-swagger issue #1088 ("editor error
// for array parameters").
//
// An array IS valid in a query parameter / response header, but its `items`
// form a Swagger 2.0 *simple schema*, which may not contain a $ref. The scanner
// used to emit `items: {$ref: #/definitions/...}`, which the Swagger editor
// rejects. The $ref must dissolve:
//   - a named primitive ([]Label, underlying string) -> items {type: string};
//   - an object element ([]Ele) has no valid simple-schema form, so it
//     dissolves to an empty items schema with a diagnostic, and must NOT leave
//     an orphan definition behind.
//
// The fix lives in the shared resolvers.ItemsTypable (now a
// schema.SimpleSchemaProbe), so the identical constraint on response headers is
// fixed by the same change.
func TestCoverage_Bug1088(t *testing.T) {
	var diagnostics []grammar.Diagnostic

	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/1088/..."},
		WorkDir:  scantest.FixturesDir(),
		OnDiagnostic: func(d grammar.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	// --- query parameters ---
	params := doc.Paths.Paths["/things"].Get.Parameters
	byName := make(map[string]int, len(params))
	for i, p := range params {
		byName[p.Name] = i
	}

	tags := params[byName["tags"]]
	require.NotNil(t, tags.Items, "tags is an array parameter")
	assert.Empty(t, tags.Items.Ref.String(),
		"a simple-schema array item must not be a $ref (go-swagger#1088)")
	assert.Equal(t, "string", tags.Items.Type,
		"a named primitive element dissolves to its underlying type")

	arr := params[byName["arr"]]
	require.NotNil(t, arr.Items)
	assert.Empty(t, arr.Items.Ref.String(),
		"an object array item must not carry a $ref in a query parameter (go-swagger#1088)")
	assert.Empty(t, arr.Items.Type,
		"an object element has no valid simple-schema form; it dissolves to empty {}")

	// --- response headers (same simple-schema constraint) ---
	headers := doc.Responses["arrayResp"].Headers
	require.Contains(t, headers, "X-Tags")
	require.Contains(t, headers, "X-Objs")

	xTags := headers["X-Tags"]
	require.NotNil(t, xTags.Items)
	assert.Empty(t, xTags.Items.Ref.String(),
		"a response-header array item must not be a $ref (go-swagger#1088)")
	assert.Equal(t, "string", xTags.Items.Type)

	xObjs := headers["X-Objs"]
	require.NotNil(t, xObjs.Items)
	assert.Empty(t, xObjs.Items.Ref.String(),
		"an object response-header array item must not carry a $ref (go-swagger#1088)")

	// --- no orphan definition ---
	// The dissolved $ref must not leave `Ele` lingering in definitions: nothing
	// references it once the array items are emptied.
	assert.NotContains(t, doc.Definitions, "Ele",
		"a dissolved simple-schema $ref must not leave an orphan definition")

	// --- diagnostics ---
	// Each unrepresentable object element emits a located diagnostic.
	var objectViolations int
	for _, d := range diagnostics {
		if d.Code != grammar.CodeUnsupportedInSimpleSchema {
			continue
		}
		assert.Equal(t, grammar.SeverityWarning, d.Severity)
		assert.NotEmpty(t, d.Pos.Filename, "diagnostic must be located")
		if strings.Contains(d.Message, "$ref") {
			objectViolations++
		}
	}
	assert.GreaterOrEqual(t, objectViolations, 2,
		"expected a %s diagnostic for each object array element (query + header)",
		grammar.CodeUnsupportedInSimpleSchema)
}
