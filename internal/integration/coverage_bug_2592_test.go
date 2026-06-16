// SPDX-FileCopyrightText: Copyright 2015-2025 go-swagger maintainers
// SPDX-License-Identifier: Apache-2.0

package integration_test

import (
	"testing"

	"github.com/go-openapi/codescan"
	"github.com/go-openapi/codescan/internal/scantest"
	"github.com/go-openapi/testify/v2/assert"
	"github.com/go-openapi/testify/v2/require"
)

// TestCoverage_Bug2592 locks the in-scope answer to go-swagger issue #2592
// ("wrap/compose a response from two types"): `// swagger:allOf` on the embedded
// components of a body type composes them into an `allOf` schema. A named
// Combined{User,Token} yields allOf:[{$ref:User},{$ref:Token}]; an inline body
// with the same embeds yields that allOf directly on the response schema.
// (Embedding in the response WRAPPER instead turns the field into a header — the
// embeds must live in the body.)
func TestCoverage_Bug2592(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages: []string{"./bugs/2592/..."}, WorkDir: scantest.FixturesDir(), ScanModels: true,
	})
	require.NoError(t, err)

	comb := doc.Definitions["Combined"]
	require.Len(t, comb.AllOf, 2)
	assert.Equal(t, "#/definitions/User", comb.AllOf[0].Ref.String())
	assert.Equal(t, "#/definitions/Token", comb.AllOf[1].Ref.String())

	inline := doc.Responses["inlineResponse"].Schema
	require.NotNil(t, inline)
	require.Len(t, inline.AllOf, 2, "inline body composes via allOf (go-swagger#2592)")
	assert.Equal(t, "#/definitions/User", inline.AllOf[0].Ref.String())
	assert.Equal(t, "#/definitions/Token", inline.AllOf[1].Ref.String())

	scantest.CompareOrDumpJSON(t, doc, "bugs_2592_schema.json")
}
