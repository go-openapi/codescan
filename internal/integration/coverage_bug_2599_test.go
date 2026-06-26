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

// TestCoverage_Bug2599 locks the resolution to go-swagger issue #2599 ("custom type on model
// fields"): a field-level `// swagger:type string` override makes a custom Go type (here UUID, an
// array under the hood) render as a bare string in the spec, without needing a strfmt type.
//
// Same mechanism answers #2404/#2419.
func TestCoverage_Bug2599(t *testing.T) {
	doc, err := codescan.Run(&codescan.Options{
		Packages:   []string{"./bugs/2599/..."},
		WorkDir:    scantest.FixturesDir(),
		ScanModels: true,
	})
	require.NoError(t, err)
	require.NotNil(t, doc)

	action := doc.Definitions["Action"]
	id := action.Properties["id"]
	require.Len(t, id.Type, 1)
	assert.Equal(t, "string", id.Type[0],
		"field-level swagger:type string overrides the UUID array type")
	assert.Empty(t, id.Ref.String(), "the override emits an inline type, not a $ref")

	scantest.CompareOrDumpJSON(t, doc, "bugs_2599_schema.json")
}
