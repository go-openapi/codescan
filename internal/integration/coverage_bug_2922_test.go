// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/spec"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2922 covers go-swagger issue #2922 ("enum description:
// superfluous name&values"): the enum const-name mapping (e.g. "FIRST
// TestEnumFirst") was unconditionally appended to the property description AND
// duplicated in x-go-enum-desc — the reporter finds the description pollution
// superfluous.
//
// Fix (decided with the maintainer): the Options knob SkipEnumDescriptions.
//   - default (false): mapping appended to the description AND exposed via
//     x-go-enum-desc — backward-compatible.
//   - true: the description is left as the authored prose; the mapping rides
//     x-go-enum-desc only.
//
// Response headers are the third OAS2 SimpleSchema target, but a special case:
// go-openapi/spec's Header.MarshalJSON does not serialize vendor extensions, so
// a header cannot carry x-go-enum-desc at all — the const mapping is simply
// unrepresentable there, in either mode. What a header CAN carry is the flat
// enum value list; a latent double-nesting bug (enum: [[FIRST, SECOND]]) in
// responseTypable.WithEnum is fixed alongside, so the values now emit correctly.
func TestCoverage_Bug2922(t *testing.T) {
	const enumMapping = "FIRST TestEnumFirst"

	t.Run("default keeps the mapping in the description (backward compat)", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./bugs/2922/..."},
			WorkDir:    scantest.FixturesDir(),
			ScanModels: true,
		})
		require.NoError(t, err)
		require.NotNil(t, doc)

		// Schema property (in: body).
		prop := doc.Definitions["GetEnumTestResponse"].Properties["testEnumInBody"]
		assert.Contains(t, prop.Description, enumMapping,
			"default behaviour: the const mapping is folded into the schema description")
		assert.Contains(t, prop.Extensions, "x-go-enum-desc")

		// Non-body parameter (in: query).
		param := queryParam(t, doc, "testEnumInQueryParam")
		assert.Contains(t, param.Description, enumMapping,
			"default behaviour: the const mapping is folded into the parameter description")
		assert.Contains(t, param.Extensions, "x-go-enum-desc")

		// Response header (in: header): flat enum, prose description, no
		// x-go-enum-desc (unrepresentable — Header doesn't marshal extensions).
		assertEnumHeader(t, doc)

		scantest.CompareOrDumpJSON(t, doc, "bugs_2922_schema.json")
	})

	t.Run("SkipEnumDescriptions leaves the description as authored prose", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:             []string{"./bugs/2922/..."},
			WorkDir:              scantest.FixturesDir(),
			ScanModels:           true,
			SkipEnumDescriptions: true,
		})
		require.NoError(t, err)
		require.NotNil(t, doc)

		// Schema property (in: body): authored prose preserved verbatim.
		prop := doc.Definitions["GetEnumTestResponse"].Properties["testEnumInBody"]
		assert.Equal(t, "The description of the test enum in the response body", prop.Description)
		assert.NotContains(t, prop.Description, enumMapping,
			"SkipEnumDescriptions: the const mapping must NOT pollute the schema description")
		schemaEnumDesc, ok := prop.Extensions.GetString("x-go-enum-desc")
		assert.True(t, ok, "the mapping must still ride x-go-enum-desc on the schema property")
		assert.True(t, strings.Contains(schemaEnumDesc, enumMapping))

		// Non-body parameter (in: query): authored prose preserved verbatim.
		param := queryParam(t, doc, "testEnumInQueryParam")
		assert.Equal(t, "The description of the test enum param", param.Description)
		assert.NotContains(t, param.Description, enumMapping,
			"SkipEnumDescriptions: the const mapping must NOT pollute the parameter description")
		paramEnumDesc, ok := param.Extensions.GetString("x-go-enum-desc")
		assert.True(t, ok, "the mapping must still ride x-go-enum-desc on the parameter")
		assert.True(t, strings.Contains(paramEnumDesc, enumMapping))

		// Response header (in: header): identical to default mode — the knob
		// does not touch headers (they never folded, and can't carry the
		// extension anyway).
		assertEnumHeader(t, doc)

		scantest.CompareOrDumpJSON(t, doc, "bugs_2922_skip_enum_desc_schema.json")
	})
}

// queryParam returns the named in=query parameter of the GET /enum/test
// operation, failing the test if it's absent.
func queryParam(t *testing.T, doc *spec.Swagger, name string) spec.Parameter {
	t.Helper()
	op := doc.Paths.Paths["/enum/test"].Get
	require.NotNil(t, op, "GET /enum/test operation must be present")
	for _, p := range op.Parameters {
		if p.Name == name && p.In == "query" {
			return p
		}
	}
	t.Fatalf("query parameter %q not found on GET /enum/test", name)
	return spec.Parameter{}
}

// assertEnumHeader checks the X-Test-Enum header of the enumHeaderResponse
// swagger:response: the enum values are a flat list (not double-nested), the
// description is the authored prose, and no x-go-enum-desc rides along (a
// header cannot serialize vendor extensions). The behaviour is identical
// regardless of SkipEnumDescriptions.
func assertEnumHeader(t *testing.T, doc *spec.Swagger) {
	t.Helper()
	resp, ok := doc.Responses["enumHeaderResponse"]
	require.True(t, ok, "enumHeaderResponse must be present in responses")
	hdr, ok := resp.Headers["X-Test-Enum"]
	require.True(t, ok, "X-Test-Enum header must be present on enumHeaderResponse")

	assert.Equal(t, []any{"FIRST", "SECOND"}, hdr.Enum,
		"header enum must be a flat value list, not double-nested")
	assert.Equal(t, "The description of the enum header", hdr.Description)
	assert.NotContains(t, hdr.Extensions, "x-go-enum-desc",
		"a response header cannot carry x-go-enum-desc (Header doesn't marshal extensions)")
}
