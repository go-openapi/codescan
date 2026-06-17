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

// TestCoverage_Bug1737 locks the answer to go-swagger issue #1737 ("bson.ObjectId
// in swagger:response generates $ref"): a field whose type resolves to a $ref
// (an external named type such as bson.ObjectId) cannot carry a sibling
// description in Swagger 2.0, so by default the field-level description is
// dropped — a $ref node has no room for it. The `DescWithRef` option is the
// supported answer: it wraps the field in `allOf: [{$ref}]` so the description
// survives alongside the reference.
func TestCoverage_Bug1737(t *testing.T) {
	t.Run("default drops the description (bare $ref)", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:   []string{"./bugs/1737/..."},
			WorkDir:    scantest.FixturesDir(),
			ScanModels: true,
		})
		require.NoError(t, err)

		someID := doc.Definitions["AResponseBody"].Properties["someId"]
		assert.Equal(t, "#/definitions/ObjectID", someID.Ref.String())
		assert.Empty(t, someID.Description,
			"a bare $ref property cannot carry a sibling description in Swagger 2.0")
	})

	t.Run("DescWithRef preserves it via allOf", func(t *testing.T) {
		doc, err := codescan.Run(&codescan.Options{
			Packages:    []string{"./bugs/1737/..."},
			WorkDir:     scantest.FixturesDir(),
			ScanModels:  true,
			DescWithRef: true,
		})
		require.NoError(t, err)

		someID := doc.Definitions["AResponseBody"].Properties["someId"]
		assert.Empty(t, someID.Ref.String(), "the $ref moves into allOf")
		require.Len(t, someID.AllOf, 1)
		assert.Equal(t, "#/definitions/ObjectID", someID.AllOf[0].Ref.String())
		assert.Equal(t, "SomeID description that the reporter wants to keep.", someID.Description)

		scantest.CompareOrDumpJSON(t, doc, "bugs_1737_schema.json")
	})
}
