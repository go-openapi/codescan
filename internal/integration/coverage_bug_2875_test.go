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

// TestCoverage_Bug2875 locks the fix for go-swagger issue #2875 ("AllOf member
// does not generate an external $ref object"): an embedded struct annotated
// `swagger:allOf` used to inline the embedded type's properties; it now emits
// a proper `allOf: [{$ref: …}]` to the embedded model's definition.
func TestCoverage_Bug2875(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2875/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	model, ok := doc.Definitions["AllOfModel"]
	require.True(t, ok)
	require.Len(t, model.AllOf, 1, "the swagger:allOf member must be an allOf arm")
	assert.Equal(t, "#/definitions/SimpleOne", model.AllOf[0].Ref.String(),
		"the allOf member must be an external $ref, not inlined properties")

	_, ok = doc.Definitions["SimpleOne"]
	assert.True(t, ok, "the embedded model must be emitted as its own definition")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2875_schema.json")
}
