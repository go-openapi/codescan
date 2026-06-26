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

// TestCoverage_Bug2038 asserts the EXPECTED resolution of go-swagger issue #2038 ("swagger ignore
// json tags for embedded structures").
//
// It is currently RED.
//
// Go's encoding/json treats an embedded struct WITH an explicit json tag as a regular named field:
// the embedded value NESTS under the tag rather than being promoted. codescan currently ignores the
// tag and flattens both forms, so the generated spec for Tagged does not match the actual JSON.
// The untagged form (correct promotion) is the green guard rail.
func TestCoverage_Bug2038(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2038/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)

	// Guard rail: untagged embed is promoted (flattened) — correct.
	untagged := doc.Definitions["Untagged"].Properties
	assert.Contains(t, untagged, "inner_field", "untagged embed promotes its fields")
	assert.Contains(t, untagged, "outer_field")

	// The #2038 defect: a json-tagged embed must NEST under the tag, not promote.
	tagged := doc.Definitions["Tagged"]
	assert.Contains(t, tagged.Properties, "outer_field")
	assert.NotContains(t, tagged.Properties, "inner_field",
		"a json-tagged embed must not promote its fields (go-swagger#2038)")
	require.Contains(t, tagged.Properties, "inner",
		"a json-tagged embed must nest under its tag (go-swagger#2038)")
	inner := tagged.Properties["inner"]
	assert.Equal(t, "#/definitions/Inner", inner.Ref.String(),
		"the nested embed should reference the embedded type")

	// `json:"-"` on an embed squashes it: Go drops it entirely, so neither the embed nor its fields
	// appear (not promoted, not nested).
	squashed := doc.Definitions["Squashed"].Properties
	assert.Contains(t, squashed, "outer_field")
	assert.NotContains(t, squashed, "inner_field", "a squashed embed promotes nothing")
	assert.NotContains(t, squashed, "inner", "a squashed embed nests nothing")
	assert.Len(t, squashed, 1)

	// swagger:allOf on a json-tagged embed: the explicit composition annotation wins over the tag —
	// the embed becomes an allOf $ref member, NOT a nested property. (Nesting fires only for plain,
	// non-allOf embeds.)
	taggedAllOf := doc.Definitions["TaggedAllOf"]
	assert.NotContains(t, taggedAllOf.Properties, "inner",
		"swagger:allOf diverts the embed away from the json-tag nesting path")
	require.Len(t, taggedAllOf.AllOf, 2)
	assert.Equal(t, "#/definitions/Inner", taggedAllOf.AllOf[0].Ref.String())
	assert.Contains(t, taggedAllOf.AllOf[1].Properties, "outer_field")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2038_schema.json")
}
